# agent-mail

MCP (Model Context Protocol) server that wraps [cloudflare_temp_email](https://github.com/dreamhunter2333/cloudflare_temp_email) API for AI agents. Multi-mailbox management, full email CRUD, search, send/reply, and advanced features.

20 MCP tools. Go binary, ~11MB, FROM scratch Docker <20MB.

## Deployment

### VPS Remote (SSE)

```bash
# 1. Create config on VPS
mkdir -p ~/.agent-mail
cat > ~/.agent-mail/config.json << 'EOF'
{
  "default_mailbox": "work",
  "mailboxes": {
    "work": {
      "name": "工作邮箱",
      "base_url": "https://mail.your-domain.com",
      "jwt": "your-address-jwt-here",
      "site_password": ""
    }
  }
}
EOF

# 2. Clone and build
git clone <repo-url> agent-mail && cd agent-mail
CGO_ENABLED=0 go build -o agent-mail .

# 3. Run on VPS (SSE on :8080)
./agent-mail --transport sse --addr :8080

# Or with Docker
docker compose up -d --build
```

### Local (stdio)

```bash
./agent-mail                          # default: --transport stdio
./agent-mail --config /path/to/config.json
```

## Transport Modes

| Flag | Mode | Use Case |
|------|------|----------|
| `--transport stdio` | stdio (default) | Local agent on same machine |
| `--transport sse` | SSE over HTTP | Remote VPS, agent connects via URL |
| `--transport streamable-http` | Streamable HTTP | Remote VPS, alternative protocol |

Use `--addr` to set listen address (default `:8080`).

## Configuration

`~/.agent-mail/config.json`:

```json
{
  "default_mailbox": "work",
  "mailboxes": {
    "work": {
      "name": "工作邮箱",
      "base_url": "https://mail.your-domain.com",
      "jwt": "eyJhbGciOiJIUzI1NiIs...",
      "site_password": ""
    }
  }
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `default_mailbox` | no | Alias of the default mailbox |
| `name` | yes | Human-readable display name |
| `base_url` | yes | API base URL of cloudflare_temp_email instance |
| `jwt` | yes | Address JWT credential (web UI → your address → credential) |
| `site_password` | no | Site-wide password if `x-custom-auth` is enabled |

## Agent Integration

### Local (stdio)

```json
{
  "mcpServers": {
    "agent-mail": {
      "command": "/path/to/agent-mail"
    }
  }
}
```

### Remote (SSE on VPS)

```json
{
  "mcpServers": {
    "agent-mail": {
      "url": "http://your-vps-ip:8080/sse"
    }
  }
}
```

### Docker (local)

```json
{
  "mcpServers": {
    "agent-mail": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "~/.agent-mail:/root/.agent-mail", "agent-mail"]
    }
  }
}
```

## Security Notes

- **VPS**: Put agent-mail behind nginx/Caddy with HTTPS + basic auth, then use `https://vps.example.com/sse` as the agent URL
- **JWT**: Never commit `config.json`. The file is mode `0600`, directory `0700`
- No auth built into the SSE endpoint — agents trust the network layer

## MCP Tools (20)

### Mailbox
| Tool | Description |
|------|-------------|
| `list_mailboxes` | List all mailboxes with JWT validity |
| `add_mailbox` | Add a mailbox credential |
| `remove_mailbox` | Remove a mailbox |
| `switch_mailbox` | Set default active mailbox |
| `validate_mailbox` | Check JWT validity |

### Read
| Tool | Description |
|------|-------------|
| `list_emails` | List received emails (paginated) |
| `get_email` | Get single email with full parsed content |
| `delete_email` | Delete a single email |
| `clear_inbox` | Delete all received emails |
| `search_emails` | Search by sender/subject keyword |

### Send
| Tool | Description |
|------|-------------|
| `send_email` | Send email from current mailbox |
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
| `list_attachments` | List S3 attachments |

## Build

```bash
go build -o agent-mail .
go test ./...        # 9 tests
```

Image: ~20MB (multi-stage, FROM scratch)
