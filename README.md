# agent-mail

MCP (Model Context Protocol) server that wraps [cloudflare_temp_email](https://github.com/dreamhunter2333/cloudflare_temp_email) API for AI agents. Supports multi-mailbox management, full email CRUD, send/reply, and advanced features.

## Quick Start

### Option 1: Docker (recommended)

```bash
# 1. Create config directory and file
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

# 2. Build and run
docker compose up -d --build
```

### Option 2: Binary

```bash
# Build
CGO_ENABLED=0 go build -o agent-mail .

# Run
./agent-mail
```

## Configuration

Location: `~/.agent-mail/config.json`

```json
{
  "default_mailbox": "work",
  "mailboxes": {
    "work": {
      "name": "工作邮箱",
      "base_url": "https://mail.your-domain.com",
      "jwt": "eyJhbGciOiJIUzI1NiIs...",
      "site_password": ""
    },
    "personal": {
      "name": "个人邮箱",
      "base_url": "https://mail.your-domain.com",
      "jwt": "eyJhbGciOiJIUzI1NiIs...",
      "site_password": "your-site-password"
    }
  }
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `default_mailbox` | no | Alias of the default mailbox |
| `name` | yes | Human-readable display name |
| `base_url` | yes | API base URL of the cloudflare_temp_email instance |
| `jwt` | yes | Address JWT credential (from web UI → your address → credential) |
| `site_password` | no | Site-wide password if deployment uses x-custom-auth |

## Agent Integration

### Hermes Agent

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

### OpenClaw

```jsonc
// Configure in OpenClaw's MCP servers config
{
  "agent-mail": {
    "command": "docker",
    "args": ["run", "-i", "--rm", "-v", "~/.agent-mail:/root/.agent-mail", "agent-mail"]
  }
}
```

## MCP Tools (20)

### Mailbox Management
| Tool | Description |
|------|-------------|
| `list_mailboxes` | List all configured mailboxes with JWT validity status |
| `add_mailbox` | Add a new mailbox credential |
| `remove_mailbox` | Remove a mailbox and its credentials |
| `switch_mailbox` | Set the default active mailbox |
| `validate_mailbox` | Check if a mailbox JWT is still valid |

### Email Read
| Tool | Description |
|------|-------------|
| `list_emails` | List received emails with pagination |
| `get_email` | Get a single email with full parsed content |
| `delete_email` | Delete a single email |
| `clear_inbox` | Delete all received emails |
| `search_emails` | Search emails by sender or subject keyword |

### Email Send
| Tool | Description |
|------|-------------|
| `send_email` | Send an email from the current mailbox |
| `check_send_balance` | Check remaining send balance |
| `list_sent` | List sent emails with pagination |
| `delete_sent` | Delete a sent email record |
| `clear_sent` | Delete all sent email records |

### Advanced
| Tool | Description |
|------|-------------|
| `get_auto_reply` | Get auto-reply configuration |
| `set_auto_reply` | Configure auto-reply settings |
| `get_webhook` | Get webhook configuration |
| `set_webhook` | Configure webhook URL |
| `list_attachments` | List S3 attachments |

## Build from Source

```bash
# Requires Go 1.22+
git clone <repo-url>
cd agent-mail
CGO_ENABLED=0 go build -o agent-mail .
./agent-mail

# Or with custom config path
./agent-mail --config /path/to/config.json
```

## Docker

```bash
docker build -t agent-mail .
docker run -i --rm -v ~/.agent-mail:/root/.agent-mail agent-mail
```

Image size: ~18MB
