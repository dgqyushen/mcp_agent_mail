# agent-mail

MCP (Model Context Protocol) server that wraps email APIs for AI agents — Cloudflare Temp Email, Gmail, QQmail. Multi-provider, multi-user, full email CRUD, send/reply, auto-reply, webhooks.

20 MCP tools. Go binary, ~11MB, FROM scratch Docker <20MB.

## Quick Start

```bash
# Build
CGO_ENABLED=0 go build -o agent-mail .

# Run (auto-creates SQLite DB at ~/.agent-mail/data.db)
./agent-mail

# With custom port and db path
./agent-mail --addr :9090 --db-path /data/agent-mail.db
```

### Docker

```bash
docker compose up -d
# Or manually:
docker run -d --name agent-mail -p 8080:8080 -v agent-mail-data:/data dgqyushen/agent-mail:latest
```

## Transport

Streamable HTTP transport only. MCP endpoint at `/mcp`.

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | Listen address |
| `--db-path` | `~/.agent-mail/data.db` | SQLite database path |
| `--env-file` | — | Path to `.env` file |
| `--auth-header` | `$AUTH_HEADER` | Legacy auth header name |
| `--auth-token` | `$AUTH_TOKEN` | Legacy auth token value |

## Authentication

MCP requests require `X-Agent-Mail-Token` header. Two modes:

### Multi-user mode (recommended)

1. Start the server, visit `http://your-server:8080/admin/login`
2. Log in with admin password (auto-generated on first run if `ADMIN_PASSWORD` env not set, check stderr logs)
3. Create users with unique tokens at `/admin/users`
4. Use the token in MCP client config

### Legacy single-user mode

Set env vars `AUTH_HEADER` and `AUTH_TOKEN`, or use CLI flags:

```bash
./agent-mail --auth-header X-Custom-Auth --auth-token my-secret

# Clients send: X-Custom-Auth: my-secret
```

## Agent Integration

### VPS (Streamable HTTP)

Put agent-mail behind nginx/Caddy with HTTPS, then:

```json
{
  "mcpServers": {
    "agent-mail": {
      "url": "https://your-domain.com/mcp",
      "headers": {
        "X-Agent-Mail-Token": "your-user-token"
      }
    }
  }
}
```

### Docker

```json
{
  "mcpServers": {
    "agent-mail": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "--network", "host", "dgqyushen/agent-mail"]
    }
  }
}
```

## Adding Mailboxes

Via web UI at `/user/login` — log in with your user token, then manage mailboxes.

Or programmatically via MCP tools: `add_mailbox`, `list_mailboxes`, `switch_mailbox`.

### Provider-specific auth_data

| Provider | `provider_type` | `auth_data` (JSON) |
|----------|----------------|-------------------|
| Cloudflare Temp Email | `cloudflare` | `{"jwt":"...","site_password":""}` |
| Gmail | `gmail` | `{"client_id":"...","client_secret":"...","refresh_token":"..."}` |
| QQmail | `qqmail` | `{"username":"...","password":"...","server":"imap.qq.com"}` |

## Web UI

| Path | Purpose |
|------|---------|
| `/` | Redirects to `/user/login` |
| `/admin/login` | Admin login |
| `/admin/users` | User management |
| `/user/login` | User self-service login |
| `/user/mailboxes` | Mailbox management |
| `/health` | Health check |

## MCP Tools (20)

### Mailbox
| Tool | Description |
|------|-------------|
| `list_mailboxes` | List all mailboxes with validation status |
| `add_mailbox` | Add a mailbox credential |
| `remove_mailbox` | Remove a mailbox |
| `switch_mailbox` | Switch active (default) mailbox |
| `validate_mailbox` | Validate mailbox accessibility |

### Read
| Tool | Description |
|------|-------------|
| `list_emails` | List received emails (paginated) |
| `get_email` | Get single email with full parsed content |
| `search_emails` | Search by sender/subject |
| `delete_email` | Delete a single email |
| `clear_inbox` | Delete all received emails |

### Send
| Tool | Description |
|------|-------------|
| `send_mail` | Send an email |
| `check_send_balance` | Check remaining send balance |
| `list_sent` | List sent emails (paginated) |
| `delete_sent` | Delete a sent email record |
| `clear_sent` | Delete all sent email records |

### Advanced
| Tool | Description |
|------|-------------|
| `get_auto_reply` | Get auto-reply config |
| `set_auto_reply` | Configure auto-reply |
| `get_webhook` | Get webhook config |
| `set_webhook` | Configure webhook URL |
| `list_attachments` | List all inbox attachments |

## Security Notes

- Put agent-mail behind nginx/Caddy with HTTPS on VPS
- Use multi-user tokens, rotate via admin UI
- Admin password is auto-generated; set `ADMIN_PASSWORD` env for persistence
- SQLite DB stored at `~/.agent-mail/data.db` with restricted permissions

## Build

```bash
go build -o agent-mail .
go test ./...
```
