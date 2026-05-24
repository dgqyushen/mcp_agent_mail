# agent-mail Redesign Spec

Date: 2026-05-24 | Status: approved

## Overview

Redesign agent-mail as a **remote-only MCP server** for AI agents to manage email. Pure streamable-http transport, SQLite storage, layered architecture with multi-provider support for future backends (Gmail, Outlook, QQ Mail, etc.).

## Architecture

```
main.go                    — CLI flags, .env loading, server startup
mcp/                       — MCP protocol layer
  server.go                — MCPServer init + tool registration
  handler.go               — 20 tool handlers (thin: validate params, call service, return result)
  tools_def.go             — Tool definitions (mcp.NewTool)
service/                   — Business logic
  mailbox_svc.go           — Mailbox CRUD, switch default, validate
  email_svc.go             — Receive: list/get/delete/clear/search
  send_svc.go              — Send: send, balance, sent items
  auto_reply_svc.go        — Auto-reply get/set
  webhook_svc.go           — Webhook get/set
  attachment_svc.go        — Attachment list
provider/                  — Email backend abstraction
  provider.go              — EmailProvider interface
  cloudflare/              — cloudflare_temp_email implementation
    client.go
store/                     — Persistence
  sqlite/
    db.go                  — Init, migrations, connection
    settings.go            — Global settings CRUD
    mailboxes.go           — Mailbox config CRUD
model/                     — Shared types
  types.go
```

**Layer dependencies:** handler → service → {provider, store}

## CLI & Configuration

```
agent-mail [flags]

--addr         :8080           Listen address
--auth-header  Authorization   Auth header name
--auth-token   ""              Auth token (empty = no auth)
--db-path      ./agent-mail.db SQLite database path
--env-file     ./.env          Env file path
```

**Loading priority (highest wins):** CLI flag → .env file → system env

**.env file:**
```
AGENT_MAIL_ADDR=:9090
AGENT_MAIL_AUTH_HEADER=X-API-Key
AGENT_MAIL_AUTH_TOKEN=my-secret-token
AGENT_MAIL_DB_PATH=/data/agent-mail.db
```

## Transport

- **Only** streamable-http (MCP reference transport)
- Single endpoint: `{addr}/mcp`
- No stdio, no SSE
- Auth middleware checks `request.Header.Get(authHeader) == authToken`, skips if token is empty

## Client Integration

```json
{
  "mcpServers": {
    "agent-mail": {
      "type": "remote",
      "url": "http://your-vps:8080/mcp",
      "enabled": true,
      "headers": {
        "X-API-Key": "my-secret-token"
      }
    }
  }
}
```

## SQLite Data Model

```sql
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE mailboxes (
    alias          TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    provider_type  TEXT NOT NULL DEFAULT 'cloudflare',
    base_url       TEXT NOT NULL,
    auth_data      TEXT NOT NULL,  -- JSON blob for provider-specific credentials
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);
```

- `auth_data` is a JSON string, allowing flexible credential schemas per provider type
- `settings` is key-value for extensibility (auth_token, default_mailbox, etc.)
- Referential integrity (`default_mailbox` → `mailboxes.alias`) enforced in application layer

## Provider Interface

```go
type EmailProvider interface {
    ListEmails(limit, offset int) (*model.PaginatedResult, error)
    GetEmail(id int) (*model.ParsedMail, error)
    DeleteEmail(id int) error
    ClearInbox() error
    SendMail(body *model.SendMailBody) error
    CheckSendBalance() (int, error)
    ListSent(limit, offset int) (*model.SendboxResult, error)
    DeleteSent(id int) error
    ClearSent() error
    GetAutoReply() (*model.AutoReplyConfig, error)
    SetAutoReply(cfg *model.AutoReplyConfig) error
    GetWebhook() (*model.WebhookSettings, error)
    SetWebhook(cfg *model.WebhookSettings) error
    ListAttachments() (*model.AttachmentListResult, error)
    Validate() error
}
```

Each provider implementation is instantiated per mailbox via `store` lookup. Adding a new backend (Gmail, Outlook, etc.) requires only implementing this interface.

## MCP Tools (20)

| Category | Tool | Description |
|----------|------|-------------|
| Mailbox | `list_mailboxes` | List all mailboxes with validity |
| Mailbox | `add_mailbox` | Add a mailbox credential |
| Mailbox | `remove_mailbox` | Remove a mailbox |
| Mailbox | `switch_mailbox` | Set default mailbox |
| Mailbox | `validate_mailbox` | Check provider connectivity |
| Read | `list_emails` | List received emails (paginated) |
| Read | `get_email` | Get single email with full content |
| Read | `delete_email` | Delete a single email |
| Read | `clear_inbox` | Delete all received emails |
| Read | `search_emails` | Search by sender/subject (client-side filter) |
| Send | `send_email` | Send email from current mailbox |
| Send | `check_send_balance` | Check remaining send balance |
| Send | `list_sent` | List sent emails (paginated) |
| Send | `delete_sent` | Delete a sent email record |
| Send | `clear_sent` | Delete all sent records |
| Advanced | `get_auto_reply` | Get auto-reply config |
| Advanced | `set_auto_reply` | Configure auto-reply |
| Advanced | `get_webhook` | Get webhook config |
| Advanced | `set_webhook` | Configure webhook URL |
| Advanced | `list_attachments` | List S3 attachments |

## Error Handling

- MCP handlers: `mcp.NewToolResultError(err.Error())`
- Service layer: wrapped errors with context (`fmt.Errorf("send: %w", err)`)
- Provider layer: raw API errors
- Non-MCP HTTP paths: `{"error": "..."}` JSON with appropriate status code

## Testing Strategy

| Layer | Approach |
|-------|----------|
| `store/sqlite` | Unit tests with `:memory:` SQLite |
| `service` | Unit tests with mocked `EmailProvider` interface |
| `mcp` | Unit tests with mocked service layer |
| `provider/cloudflare` | Optional integration tests with fake HTTP server |

## Docker

```dockerfile
FROM golang:1.25-alpine AS build
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -o agent-mail .

FROM scratch
COPY --from=build /app/agent-mail /
ENTRYPOINT ["/agent-mail"]
```

```bash
docker run -v /data:/data agent-mail --db-path /data/agent-mail.db --env-file /data/.env
```

## What's Removed

- `--transport` flag (stdio, SSE)
- `--config` flag and TOML config system
- `config/` package
- `model.DefaultConfigPath`

## Migration from Old Config

Migration script or manual import: read old `config.toml`, use `add_mailbox` MCP tool to populate new SQLite store. (Out of scope for initial implementation — addressed as follow-up.)
