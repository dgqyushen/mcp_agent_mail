# User Self-Service Portal Design

## Summary

Let end-users log into a web frontend with their `atm-xxx` token and manage their own email provider configurations (mailboxes). The existing admin panel (`/admin/`) remains unchanged. The user's entry points are at `/user/`.

## Motivation

Currently only the admin can create users and generate tokens. Users themselves cannot view or configure their mailboxes (providers, credentials) except through the MCP API (`add_mailbox`, etc.). This adds a browser-based self-service UI.

## Design

### Routes

| Path | Purpose |
|------|---------|
| `GET /user/login` | Show token login form |
| `POST /user/login` | Validate token, create session, redirect to `/user/mailboxes` |
| `GET /user/mailboxes` | List user's mailboxes with validation status |
| `GET /user/mailboxes/add` | Show add-mailbox form (provider selector + dynamic fields) |
| `POST /user/mailboxes/add` | Create new mailbox |
| `GET /user/mailboxes/edit?alias=xxx` | Show edit form for existing mailbox |
| `POST /user/mailboxes/edit` | Update mailbox fields |
| `POST /user/mailboxes/delete` | Delete a mailbox |
| `POST /user/logout` | Clear session |

### Authentication Flow

1. User visits `/user/login`, enters their `atm-xxx` token
2. Server calls `UserService.ValidateToken(token)` → returns `userID`
3. Server creates a session cookie `user_session` (path `/user`, HttpOnly, SameSite Strict)
4. Session is in-memory, stores `(userID, expiry)` with 12h TTL
5. Subsequent requests under `/user/` check the cookie and extract `userID`

### Session Implementation

Extend `web/session.go`:
- Separate cookie name: `user_session`, path: `/user`
- Separate in-memory map: `userSessions map[string]userSessionData`
- `setUserSession(w, userID) string`
- `getUserSession(r) (int, bool)` — returns userID, valid
- `clearUserSession(w)`

### Mailbox Management

All mailbox operations reuse existing services:
- `MailboxService.List(userID)` — list with validation status
- `MailboxService.Add(userID, alias, name, providerType, baseURL, authData)`
- `MailboxService.Remove(userID, alias)`
- `MailboxService.Switch(userID, alias)` — set default
- `MailboxService.Validate(userID, alias)` — validate and return settings

### Provider Form Fields

Each provider registers structured field metadata so the frontend can render appropriate form inputs:

```go
type FieldDef struct {
    Key      string // field name in auth_data JSON
    Label    string // display label
    Type     string // "text" or "password"
    Section  string // "base_url" or "auth_data"
}

type ProviderFormInfo struct {
    Type   string     // "cloudflare", "gmail", "qqmail"
    Label  string     // display name e.g. "Cloudflare"
    Fields []FieldDef
}
```

Provider-specific field definitions:

| Provider | base_url | auth_data fields |
|----------|----------|-----------------|
| cloudflare | `base_url` (text, required) | `jwt` (password, required), `site_password` (password, optional) |
| gmail | (none) | `client_id` (text, required), `client_secret` (password, required), `access_token` (text, required), `refresh_token` (text, required), `token_expiry` (text, required) |
| qqmail | (none) | `username` (text, required), `password` (password, required) |

The form page includes a small JavaScript snippet to show/hide field groups based on the selected provider type.

### Templates (new)

- `user_login.html` — Token login form (simple centered card, one text input + submit)
- `user_mailboxes.html` — Mailbox list table with add/edit/delete actions + validation status
- `user_mailbox_form.html` — Shared add/edit form with provider selector and dynamic fields

### File Changes

| File | Change |
|------|--------|
| `provider/factory.go` | Add `FieldDef`, `ProviderFormInfo` types + `GetProviderFormInfo()` + `RegisterProviderFormInfo()` |
| `provider/cloudflare/provider.go` | Register form fields for cloudflare |
| `provider/gmail/provider.go` | Register form fields for gmail |
| `provider/qqmail/provider.go` | Register form fields for qqmail |
| `web/session.go` | Add user session support (`user_session` cookie) |
| `web/handler.go` | Add user route handlers |
| `web/templates/user_login.html` | New |
| `web/templates/user_mailboxes.html` | New |
| `web/templates/user_mailbox_form.html` | New |
| `web/templates/base.html` | Minor: add login context to header (show user name or admin link) |
| `main.go` | Pass `mailboxSvc` to `NewAdminHandler` (so user routes can use it) |
| `mcp/tools_test.go` | May need update if web handler interface changes |

### What Stays Unchanged

- Admin panel (`/admin/`): all routes, templates, and behavior
- MCP API: all tools and auth
- Database schema
- Provider factory and provider implementations
- Tests for existing packages (new tests for new code only)

### Constraints

- All session data is in-memory (server restart logs everyone out) — matches admin behavior
- Cookie is HttpOnly, SameSite=Strict, path-scoped to `/user` or `/admin`
- Provider auth_data is stored as-is in the database (JSON string in `auth_data` column)

## Open Questions (resolved)

- Q: Separate user UI or extend admin? → Extend the same Go server, separate path prefix
- Q: Full email management or just mailbox config? → Just mailbox config
- Q: Go html/template or SPA? → Go html/template
- Q: Generic JSON field or structured forms? → Structured forms per provider type
