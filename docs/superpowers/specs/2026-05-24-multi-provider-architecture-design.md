# Multi-Provider Architecture Design

## Goal

Abstract email receiving and sending into separate interfaces, enabling Gmail and QQmail backends alongside the existing Cloudflare Temp Mail provider. Users configure providers via `add_mailbox` MCP tool with provider-specific credentials.

## Architecture

### Interfaces

```go
// provider/receiver.go
type MailReceiver interface {
    GetSettings() (*model.SettingsResponse, error)
    ListEmails(limit, offset int) (*model.PaginatedResult, error)
    GetEmail(id int) (*model.ParsedMail, error)
    DeleteEmail(id int) error
    ClearInbox() error
    GetAutoReply() (*model.AutoReplyConfig, error)
    SetAutoReply(*model.AutoReplyConfig) error
    GetWebhook() (*model.WebhookSettings, error)
    SetWebhook(*model.WebhookSettings) error
    ListAttachments() (*model.AttachmentListResult, error)
    Validate() error
}

// provider/sender.go
type MailSender interface {
    SendMail(*model.SendMailBody) error
    CheckSendBalance() (int, error)
    ListSent(limit, offset int) (*model.SendboxResult, error)
    DeleteSent(id int) error
    ClearSent() error
    Validate() error
}
```

### Provider Struct

```go
// provider/provider.go
type MailProvider struct {
    Record   model.MailboxRecord
    Receiver MailReceiver
    Sender   MailSender
}

var ErrCapNotSupported = errors.New("capability not supported by this provider")
```

`Receiver` or `Sender` may be nil. Service layer checks nil and returns `ErrCapNotSupported`.

### Factory Registration

```go
type ProviderFactory interface {
    NewProvider(record model.MailboxRecord) (*MailProvider, error)
}
```

Switch on `record.ProviderType`: "cloudflare", "gmail", "qqmail".

## Directory Structure

```
provider/
├── provider.go          # MailProvider struct, errors, common helpers
├── receiver.go          # MailReceiver interface
├── sender.go            # MailSender interface
├── factory.go           # DefaultProviderFactory
├── cloudflare/
│   ├── client.go        # Cloudflare REST client → MailReceiver + MailSender
│   └── provider.go      # NewProvider(record) → *MailProvider
├── gmail/
│   ├── provider.go      # NewProvider(record) → *MailProvider
│   ├── receiver.go      # Gmail API implementation
│   └── sender.go        # SMTP + XOAUTH2 implementation
└── qqmail/
    ├── provider.go      # NewProvider(record) → *MailProvider
    ├── receiver.go      # IMAP implementation
    └── sender.go        # SMTP + LOGIN implementation
```

## Provider Backends

### Cloudflare (existing, minimal change)

| Capability | Implementation |
|-----------|---------------|
| Receiver | Existing `/api/parsed_mails`, `/api/settings`, `/api/auto_reply`, `/api/webhook/settings`, `/api/attachment/list` |
| Sender | Existing `/api/send_mail`, `/api/sendbox` |
| Auth | `auth_data: {"jwt":"...","site_password":"..."}` + `base_url` |

Changes: Split existing `Client` struct to implement `MailReceiver` and `MailSender` interfaces. No logic change.

### Gmail

**New dependencies**: `google.golang.org/api/gmail/v1`, `golang.org/x/oauth2`

| Method | Implementation |
|--------|---------------|
| GetSettings | `users.getProfile` → email address, no send balance concept |
| ListEmails | `users.messages.list` with pagination |
| GetEmail | `users.messages.get` with format=full |
| DeleteEmail | `users.messages.trash` (Gmail soft delete) |
| ClearInbox | `users.messages.batchDelete` with all message IDs |
| SearchEmails | `users.messages.list` with `q=` parameter (server-side) |
| GetAutoReply | `users.settings.vacation.get` |
| SetAutoReply | `users.settings.vacation.set` |
| GetWebhook | Return `ErrCapNotSupported` |
| SetWebhook | Return `ErrCapNotSupported` (requires Google Pub/Sub) |
| ListAttachments | Parse attachments from `messages.get` payload parts |
| SendMail | `net/smtp` to `smtp.gmail.com:587` with XOAUTH2 |
| CheckSendBalance | Return `(0, ErrCapNotSupported)` |
| ListSent | `users.messages.list` with `q=in:sent` |
| DeleteSent | `users.messages.trash` |
| ClearSent | `users.messages.list in:sent` + batch trash |
| Validate | Verify token works via `users.getProfile` |

**Auth data**:
```json
{
    "access_token": "ya29...",
    "refresh_token": "1//0g...",
    "client_id": "xxx.apps.googleusercontent.com",
    "client_secret": "GOCSPX-...",
    "token_expiry": "2026-05-24T10:00:00Z"
}
```

Token auto-refresh: If server detects expired access_token, uses refresh_token + client_id/secret to obtain new one.

### QQmail

**New dependencies**: `github.com/emersion/go-imap` v2

| Method | Implementation |
|--------|---------------|
| GetSettings | IMAP LIST → identify INBOX, SMTP validate → return email from auth_data |
| ListEmails | IMAP SELECT INBOX + FETCH 1:* (FLAGS) + paginate |
| GetEmail | IMAP FETCH BODY[] + MIME parse |
| DeleteEmail | IMAP STORE +FLAGS.SILENT (\Deleted) + EXPUNGE |
| ClearInbox | IMAP SEARCH ALL + STORE (\Deleted) + EXPUNGE |
| SearchEmails | IMAP SEARCH (FROM/SUBJECT) |
| GetAutoReply | Return `ErrCapNotSupported` |
| SetAutoReply | Return `ErrCapNotSupported` |
| GetWebhook | Return `ErrCapNotSupported` |
| SetWebhook | Return `ErrCapNotSupported` |
| ListAttachments | MIME multipart parse from FETCH BODY[] |
| SendMail | `net/smtp` to `smtp.qq.com:465` with LOGIN |
| CheckSendBalance | Return `(0, ErrCapNotSupported)` |
| ListSent | IMAP LIST to discover Sent folder, then SELECT + FETCH |
| DeleteSent | Same as DeleteEmail on Sent folder |
| ClearSent | Same as ClearInbox on Sent folder |
| Validate | IMAP login + LOGOUT success check |

**Auth data**:
```json
{
    "username": "xxx@qq.com",
    "password": "授权码"
}
```

Note: QQmail IMAP server `imap.qq.com:993` (SSL). SMTP server `smtp.qq.com:465` (SSL).

## Service Layer Impact

Only `service/mailbox_svc.go` changes:

```go
func (s *MailboxService) Provider(userID int, alias string) (*provider.MailProvider, error)
func (s *MailboxService) Receiver(userID int, alias string) (provider.MailReceiver, error) {
    p, err := s.Provider(userID, alias)
    if err != nil { return nil, err }
    if p.Receiver == nil { return nil, provider.ErrCapNotSupported }
    return p.Receiver, nil
}
func (s *MailboxService) Sender(userID int, alias string) (provider.MailSender, error) {
    p, err := s.Provider(userID, alias)
    if err != nil { return nil, err }
    if p.Sender == nil { return nil, provider.ErrCapNotSupported }
    return p.Sender, nil
}
```

Other services (email, send, auto_reply, webhook, attachment) update their calls to use `s.mailboxSvc.Receiver()` or `s.mailboxSvc.Sender()`.

## MCP Tool Changes

- `add_mailbox`: `provider_type` description updated. `auth_data` description updated to list per-provider schemas.
- Existing tool interfaces unchanged.

## Config Changes

None. New providers are self-contained packages. No new CLI flags or env vars.

## Testing Strategy

- **Cloudflare**: Existing tests continue working. No fundamental change.
- **Gmail/QQmail**: No real HTTP/IMAP calls in unit tests. Use `testProviderFactory` pattern (already exists in tests) to inject mock providers.
- Integration tests (optional): Use real Gmail/QQmail credentials via env vars, gated behind build tags.

## Implementation Phases

Given the scope, implement in 3 phases:

**Phase 1 — Interface Refactor**: Extract `MailReceiver`/`MailSender` interfaces, create `MailProvider` struct, update factory switch. Refactor existing Cloudflare provider to implement both interfaces. Update service layer. Tests pass. No new dependencies.

**Phase 2 — Gmail Backend**: Add `google.golang.org/api/gmail/v1` + `golang.org/x/oauth2`. Implement Gmail receiver (Gmail API) + sender (SMTP XOAUTH2). Unit tests with mock.

**Phase 3 — QQmail Backend**: Add `github.com/emersion/go-imap`. Implement QQmail receiver (IMAP) + sender (SMTP LOGIN). Unit tests with mock.

Each phase is independently testable and deployable.

## Future Considerations

- **Provider registry pattern**: If more backends added later, consider `init()` registration instead of switch in factory.
- **OAuth flow**: Future `mcp` tool could host a local callback server to streamline Gmail OAuth token acquisition.
- **IMAP IDLE**: Future enhancement for QQmail real-time email push.
