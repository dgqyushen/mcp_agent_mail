# AGENTS.md

## Build & Test

```bash
go build -o agent-mail .    # binary
go test ./...                # all tests (9)
go test ./mcp                # single package
```

No linter, formatter, or CI configured.

## Architecture

```
main.go          — entrypoint, CLI flags, transport dispatch
config/          — TOML config Load/Save (~/.agent-mail/config.toml)
client/          — HTTP client wrapping cloudflare_temp_email API
mcp/             — MCP server: 20 tool handlers + tool definitions
model/           — shared types and DefaultConfigPath
```

- **Stdout is reserved** for MCP protocol (stdio transport). All logging/errors go to **stderr** via `log/slog`.
- Config default path: `~/.agent-mail/config.toml` (auto-created `0600`, dir `0700`).

## Transport Modes

| Flag | Mode | Use |
|---|---|---|
| `--transport stdio` | default | local agent |
| `--transport sse` | SSE/HTTP | remote VPS |
| `--transport streamable-http` | Streamable HTTP | remote VPS |

`--addr` sets listen address (default `:8080`), only used for HTTP-based transports.

## Docker

Static binary: `CGO_ENABLED=0 GOOS=linux go build -o agent-mail .` then `FROM scratch`.

## Testing Notes

- `mcp` tests use fake placeholder API URLs — no real HTTP calls.
- `model.DefaultConfigPath` is a package-level `var func()` that can be overridden in tests.
- `config` tests write to `t.TempDir()` and test TOML round-trip.

## Logging

Uses `log/slog` (Go stdlib). Logger set via `slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))` in `main.go`. Key client HTTP requests log at `Debug` level, errors at `Error`.

## Authentication

Bearer token auth for HTTP transports. Token source priority:

1. `--auth-token` CLI flag (highest)
2. `auth_token` in `config.toml`
3. If neither set, no auth (open access)

Config TOML example:
```toml
auth_token = "your-secret-token"
```

Clients send `Authorization: Bearer <token>` header. Unauthenticated requests get 401 JSON response. stdio transport always skips auth.
