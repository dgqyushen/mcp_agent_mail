# AGENTS.md

## Build & Test

```bash
go build -o agent-mail .    # binary
go test ./...                # all tests
go test ./mcp                # single package
```

No linter, formatter, or CI configured.

## Architecture

```
main.go          — entrypoint, CLI flags, server bootstrap
mcp/             — MCP server: 20 tool handlers + tool definitions + auth middleware
model/           — shared types (ParsedMail, MailboxInfo, etc.)
provider/        — email provider abstraction (cloudflare/, gmail/, qqmail/)
service/         — business logic (mailbox, email, send, auto-reply, webhook, attachment, user)
store/sqlite/    — SQLite persistence (users, tokens, mailboxes, settings)
web/             — embedded web UI (admin panel + user self-service portal)
```

- All logging/errors go to **stderr** via `log/slog`.
- Database: SQLite at `~/.agent-mail/data.db` (default), auto-created with migrations.

## Transport

Streamable HTTP only. MCP endpoint at `/mcp`:

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | Listen address |
| `--db-path` | `~/.agent-mail/data.db` | SQLite database path |
| `--env-file` | — | Path to `.env` file for env vars |
| `--auth-header` | `$AUTH_HEADER` | Legacy custom auth header name |
| `--auth-token` | `$AUTH_TOKEN` | Legacy auth token value |

## Docker

Multi-stage build: `golang:1.25-alpine` builder → `FROM scratch`. CGO disabled.

```bash
CGO_ENABLED=0 GOOS=linux go build -o agent-mail .
docker build -t agent-mail .
```

Exposes port 8080, volume at `/data`, entrypoint with `--db-path /data/agent-mail.db`.

## Testing Notes

- `mcp` tests use in-memory SQLite and fake placeholder API URLs — no real HTTP calls.
- Provider tests (cloudflare, gmail, qqmail) test validation/building logic without real network I/O.
- `web` tests cover session management, CSRF, and auth middleware.

## Logging

Uses `log/slog` (Go stdlib). Logger set via `slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))` in `main.go`.

## Authentication

MCP endpoint requires `X-Agent-Mail-Token` header. Two token sources:

1. **User token** (multi-user): Created via admin web UI at `/admin`. Each user gets a unique token; validated against SQLite.
2. **Legacy token** (single-user): Set via `AUTH_HEADER` + `AUTH_TOKEN` env vars. Bypasses SQLite auth (assigned user_id=0).

Unauthenticated requests get `401` JSON response.

## Web UI

| Path | Purpose |
|------|---------|
| `/` | Redirects to `/user/login` |
| `/admin/login` | Admin login (password from `ADMIN_PASSWORD` env or auto-generated) |
| `/admin/users` | User management (create, list, refresh token, delete) |
| `/user/login` | User self-service login with token |
| `/user/mailboxes` | User mailbox management (add, edit, delete, switch) |
| `/health` | Health check endpoint |

Admin password is auto-generated on first run if `ADMIN_PASSWORD` env is not set (bcrypt hashed, stored in SQLite).
