package mcp

import (
	mcp "github.com/mark3labs/mcp-go/mcp"
)

const (
	ToolListMailboxes   = "list_mailboxes"
	ToolAddMailbox      = "add_mailbox"
	ToolRemoveMailbox   = "remove_mailbox"
	ToolSwitchMailbox   = "switch_mailbox"
	ToolValidateMailbox = "validate_mailbox"
	ToolListEmails      = "list_emails"
	ToolGetEmail        = "get_email"
	ToolSearchEmails    = "search_emails"
	ToolDeleteEmail     = "delete_email"
	ToolClearInbox      = "clear_inbox"
	ToolSendMail        = "send_mail"
	ToolCheckBalance    = "check_send_balance"
	ToolListSent        = "list_sent"
	ToolDeleteSent      = "delete_sent"
	ToolClearSent       = "clear_sent"
	ToolGetAutoReply    = "get_auto_reply"
	ToolSetAutoReply    = "set_auto_reply"
	ToolGetWebhook      = "get_webhook"
	ToolSetWebhook      = "set_webhook"
	ToolListAttach      = "list_attachments"
)

var Tools = []mcp.Tool{
	// Mailbox management
	{
		Name:        ToolListMailboxes,
		Description: "List all configured mailboxes and their validation status",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	},
	{
		Name:        ToolAddMailbox,
		Description: "Add a new email mailbox to the configuration",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias":         map[string]any{"type": "string", "description": "Short alias for the mailbox"},
				"name":          map[string]any{"type": "string", "description": "Display name for the mailbox"},
				"provider_type": map[string]any{"type": "string", "description": "Provider type (cloudflare, future: gmail, outlook, qq)"},
				"base_url":      map[string]any{"type": "string", "description": "API base URL for the email service"},
				"auth_data":     map[string]any{"type": "string", "description": "JSON string with auth credentials e.g. {\"jwt\":\"...\",\"site_password\":\"...\"}"},
			},
			Required: []string{"alias", "name", "base_url", "auth_data"},
		},
	},
	{
		Name:        ToolRemoveMailbox,
		Description: "Remove a configured mailbox",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox to remove"},
			},
			Required: []string{"alias"},
		},
	},
	{
		Name:        ToolSwitchMailbox,
		Description: "Switch the active (default) mailbox",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox to switch to"},
			},
			Required: []string{"alias"},
		},
	},
	{
		Name:        ToolValidateMailbox,
		Description: "Validate that a mailbox is accessible and returns its settings",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox to validate (defaults to active mailbox)"},
			},
		},
	},
	// Email operations
	{
		Name:        ToolListEmails,
		Description: "List emails in the inbox",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias":  map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"limit":  map[string]any{"type": "integer", "description": "Maximum number of emails to return (default 20)"},
				"offset": map[string]any{"type": "integer", "description": "Offset for pagination (default 0)"},
			},
		},
	},
	{
		Name:        ToolGetEmail,
		Description: "Get full details of a specific email by ID",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"id":    map[string]any{"type": "integer", "description": "Email ID"},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        ToolSearchEmails,
		Description: "Search emails by sender or subject (client-side search, fetches latest 100 emails)",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"query": map[string]any{"type": "string", "description": "Search query (matches against sender and subject)"},
				"limit": map[string]any{"type": "integer", "description": "Maximum results (default 20)"},
			},
			Required: []string{"query"},
		},
	},
	{
		Name:        ToolDeleteEmail,
		Description: "Delete a specific email by ID",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"id":    map[string]any{"type": "integer", "description": "Email ID to delete"},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        ToolClearInbox,
		Description: "Delete all emails in the inbox",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
			},
		},
	},
	// Send operations
	{
		Name:        ToolSendMail,
		Description: "Send an email",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias":     map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"from_name": map[string]any{"type": "string", "description": "Sender display name"},
				"to_mail":   map[string]any{"type": "string", "description": "Recipient email address"},
				"to_name":   map[string]any{"type": "string", "description": "Recipient display name"},
				"subject":   map[string]any{"type": "string", "description": "Email subject"},
				"content":   map[string]any{"type": "string", "description": "Email body content"},
				"is_html":   map[string]any{"type": "boolean", "description": "Whether content is HTML"},
			},
			Required: []string{"to_mail", "subject", "content"},
		},
	},
	{
		Name:        ToolCheckBalance,
		Description: "Check remaining email send balance",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
			},
		},
	},
	{
		Name:        ToolListSent,
		Description: "List sent emails",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias":  map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"limit":  map[string]any{"type": "integer", "description": "Maximum number (default 20)"},
				"offset": map[string]any{"type": "integer", "description": "Offset for pagination (default 0)"},
			},
		},
	},
	{
		Name:        ToolDeleteSent,
		Description: "Delete a specific sent email by ID",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"id":    map[string]any{"type": "integer", "description": "Sent email ID to delete"},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        ToolClearSent,
		Description: "Delete all sent emails",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
			},
		},
	},
	// Auto-reply
	{
		Name:        ToolGetAutoReply,
		Description: "Get current auto-reply configuration",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
			},
		},
	},
	{
		Name:        ToolSetAutoReply,
		Description: "Set auto-reply configuration",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias":         map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"name":          map[string]any{"type": "string", "description": "Auto-reply name"},
				"subject":       map[string]any{"type": "string", "description": "Auto-reply subject"},
				"source_prefix": map[string]any{"type": "string", "description": "Source prefix filter"},
				"message":       map[string]any{"type": "string", "description": "Auto-reply message body"},
				"enabled":       map[string]any{"type": "boolean", "description": "Enable or disable auto-reply"},
			},
			Required: []string{"enabled"},
		},
	},
	// Webhook
	{
		Name:        ToolGetWebhook,
		Description: "Get current webhook configuration",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
			},
		},
	},
	{
		Name:        ToolSetWebhook,
		Description: "Set webhook configuration",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias":  map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
				"url":    map[string]any{"type": "string", "description": "Webhook URL"},
				"events": map[string]any{"type": "array", "description": "List of event types to subscribe to", "items": map[string]any{"type": "string"}},
			},
			Required: []string{"url", "events"},
		},
	},
	// Attachments
	{
		Name:        ToolListAttach,
		Description: "List all attachments in the inbox",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"alias": map[string]any{"type": "string", "description": "Alias of the mailbox (defaults to active)"},
			},
		},
	},
}
