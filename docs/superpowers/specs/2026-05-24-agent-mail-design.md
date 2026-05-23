# Agent-Mail MCP Server Design

## Overview

A Go-based MCP (Model Context Protocol) server that wraps the [cloudflare_temp_email](https://github.com/dreamhunter2333/cloudflare_temp_email) API, enabling AI agents (Hermes Agent, OpenClaw, etc.) to manage multiple email mailboxes with rich CRUD operations.

## Goals

- Multi-mailbox management with credential persistence
- Full email CRUD: list, search, read, delete, send
- Sendbox, auto-reply, webhook, and attachment management
- Single binary, minimal resource footprint (sub-15MB Docker image)
- Deployable on a VPS via Docker

## Non-Goals

- Local email storage or offline mode — all data is live via API
- Web UI or CLI for human use
- Mailbox creation — users create mailboxes in the temp-mail web UI
- Support for non-cloudflare_temp_email backends

---

## Architecture

```
+---------------------+
|  Hermes / OpenClaw  |---- MCP stdio ----+
+---------------------+                   |
                                   +-------v----------+
                                   |  agent-mail       |
                                   |  (Go MCP Server)  |
                                   |                   |
                                   |  +-------------+  |
                                   |  | Config      |  |
                                   |  | Manager     |  |
                                   |  +-------------+  |
                                   |  +-------------+  |
                                   |  | API Client  |  |
                                   |  +-------------+  |
                                   +-------+-----------+
                                           | HTTPS
                              +------------+------------+
                              v            v            v
                         +--------+  +--------+  +--------+
                         |mailbox |  |mailbox |  |mailbox |
                         |   A    |  |   B    |  |   C    |
                         +--------+  +--------+  +--------+
                        (cloudflare_temp_email instances)
```

### Project Structure

```
agent-mail/
  main.go              # Entry point, MCP server init
  config/
    config.go          # Read/write ~/.agent-mail/config.json
  mcp/
    tools.go           # MCP tool definitions and handlers
  client/
    api.go             # HTTP client for cloudflare_temp_email API
  model/
    types.go           # Shared data structures
  go.mod / go.sum
  Dockerfile
```

### Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go 1.22+ | Single binary, minimal memory |
| MCP SDK | `github.com/mark3labs/mcp-go` | Most mature Go MCP library |
| HTTP Client | `net/http` (stdlib) | No external dep needed |
| JSON | `encoding/json` (stdlib) | Zero-dependency |
| Config | `~/.agent-mail/config.json` | Simple, human-editable |
| Docker | `FROM scratch` | Minimal image |

---

## Configuration

### Config File: `~/.agent-mail/config.json`

```json
{
  "default_mailbox": "work",
  "mailboxes": {
    "work": {
      "name": "工作邮箱",
      "base_url": "https://mail.example.com",
      "jwt": "eyJ...",
      "site_password": ""
    }
  }
}
```

- `default_mailbox` — the mailbox used when no explicit `mailbox` arg is given
- `mailboxes` — keyed by a short alias; each entry has the API base URL, address JWT, and optional site password

### Config Path

- Linux/macOS: `~/.agent-mail/config.json`
- Docker: mounted at `/root/.agent-mail/config.json` via volume

---

## MCP Tools

### Mailbox Management (5 tools)

| Tool | Args | Returns | Description |
|------|------|---------|-------------|
| `list_mailboxes` | — | `[{alias, name, base_url, valid}]` | List all configured mailboxes with validity |
| `add_mailbox` | `alias, name, base_url, jwt, site_password?` | `{success}` | Add a new mailbox credential |
| `remove_mailbox` | `alias` | `{success}` | Remove a mailbox |
| `switch_mailbox` | `alias` | `{success}` | Set default active mailbox |
| `validate_mailbox` | `alias?` | `{address, send_balance}` | Check JWT validity via `/api/settings` |

### Email Read (4 tools)

| Tool | Args | Returns | Description |
|------|------|---------|-------------|
| `list_emails` | `mailbox?, limit?, offset?` | `[{id, sender, subject, created_at}]` | List inbox with pagination |
| `get_email` | `mail_id, mailbox?` | `{id, sender, subject, text, html, attachments}` | Full email content via `/api/parsed_mail/:id` |
| `delete_email` | `mail_id, mailbox?` | `{success}` | Delete a single email |
| `clear_inbox` | `mailbox?` | `{success}` | Delete all inbox emails |

### Email Search (1 tool)

| Tool | Args | Returns | Description |
|------|------|---------|-------------|
| `search_emails` | `query, mailbox?, limit?` | `[{id, sender, subject, created_at}]` | Client-side search: fetches up to 100 recent emails and filters by sender/subject containing query |

### Email Send (5 tools)

| Tool | Args | Returns | Description |
|------|------|---------|-------------|
| `send_email` | `to_mail, subject, content, mailbox?, from_name?, to_name?, is_html?` | `{status}` | Send an email |
| `check_send_balance` | `mailbox?` | `{balance}` | Get remaining send quota |
| `list_sent` | `mailbox?, limit?, offset?` | `[{id, to, subject, created_at}]` | List sent emails |
| `delete_sent` | `send_id, mailbox?` | `{success}` | Delete a sent email record |
| `clear_sent` | `mailbox?` | `{success}` | Clear all sent records |

### Advanced (5 tools)

| Tool | Args | Returns | Description |
|------|------|---------|-------------|
| `get_auto_reply` | `mailbox?` | `{subject, message, enabled, name}` | Get auto-reply settings |
| `set_auto_reply` | `mailbox?, subject?, message?, enabled?, name?` | `{success}` | Update auto-reply settings |
| `get_webhook` | `mailbox?` | `{url, events}` | Get webhook config |
| `set_webhook` | `mailbox?, url, events?` | `{success}` | Configure webhook |
| `list_attachments` | `mailbox?` | `[{key}]` | List S3 attachments for current address |

**Total: 20 MCP tools**

---

## API Client Design

The HTTP client wraps all cloudflare_temp_email API calls:

- Base URL + JWT per mailbox from config
- Required headers per request: `Authorization: Bearer <JWT>`, optional `x-custom-auth`, `x-lang: en`
- Error handling: map HTTP errors to structured Go errors
- Rate limiting: respect `429` with exponential backoff
- Timeout: 30s default, configurable per call

### Supported Endpoints

| Client Method | API Endpoint | Used By |
|---------------|-------------|---------|
| `GetSettings` | `GET /api/settings` | `validate_mailbox`, `check_send_balance` |
| `ListParsedMails` | `GET /api/parsed_mails` | `list_emails` |
| `GetParsedMail` | `GET /api/parsed_mail/:id` | `get_email` |
| `DeleteMail` | `DELETE /api/mails/:id` | `delete_email` |
| `ClearInbox` | `DELETE /api/clear_inbox` | `clear_inbox` |
| `SendMail` | `POST /api/send_mail` | `send_email` |
| `ListSendbox` | `GET /api/sendbox` | `list_sent` |
| `DeleteSendbox` | `DELETE /api/sendbox/:id` | `delete_sent` |
| `ClearSentItems` | `DELETE /api/clear_sent_items` | `clear_sent` |
| `GetAutoReply` | `GET /api/auto_reply` | `get_auto_reply` |
| `SetAutoReply` | `POST /api/auto_reply` | `set_auto_reply` |
| `GetWebhook` | `GET /api/webhook/settings` | `get_webhook` |
| `SetWebhook` | `POST /api/webhook/settings` | `set_webhook` |
| `ListAttachments` | `GET /api/attachment/list` | `list_attachments` |

---

## Error Handling

- **Config errors**: file not found, malformed JSON — MCP tool returns descriptive error
- **Auth errors**: JWT expired (401) — update config validity flag, return `"JWT expired, please re-add mailbox"`
- **Network errors**: timeout, connection refused — retry once, then return error
- **Rate limits**: 429 — sleep 3s, retry once, then return error
- **Parse errors**: API returned unexpected format — return raw error for agent to handle

---

## Docker Deployment

```dockerfile
FROM scratch
COPY agent-mail /agent-mail
ENTRYPOINT ["/agent-mail"]
```

```yaml
# docker-compose.yml
services:
  agent-mail:
    build: .
    volumes:
      - ~/.agent-mail:/root/.agent-mail
    restart: unless-stopped
```

The agent invokes the binary directly via MCP stdio transport — no ports exposed. Docker is purely for isolation and auto-restart.

---

## Testing

- Unit tests for config parsing, API client, MCP tool handlers
- Integration tests with a mock HTTP server simulating cloudflare_temp_email API
- `go test ./...` in CI

---

## Future Considerations (out of scope for v1)

- `mailbox` parameter could accept comma-separated list for cross-mailbox search
- Attachment download support
- Admin API integration for full mailbox lifecycle management
- WebSocket or SSE-based new-mail notifications
