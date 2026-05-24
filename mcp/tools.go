package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"agent-mail/client"
	"agent-mail/config"
	"agent-mail/model"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	mcpServer *server.MCPServer
	cfg       *model.Config
	cfgPath   string
	mu        sync.Mutex
}

func New(cfg *model.Config, cfgPath string) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer("agent-mail", "1.0.0"),
		cfg:       cfg,
		cfgPath:   cfgPath,
	}
	s.registerTools()
	return s
}

func (s *Server) getClientForMailbox(alias string) (*client.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if alias == "" {
		alias = s.cfg.DefaultMailbox
	}
	mb, ok := s.cfg.Mailboxes[alias]
	if !ok {
		return nil, fmt.Errorf("mailbox %q not found", alias)
	}
	return client.New(mb.BaseURL, mb.JWT, mb.SitePassword), nil
}

func (s *Server) saveConfig() error {
	return config.Save(s.cfgPath, s.cfg)
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) handleListMailboxes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var infos []model.MailboxInfo
	for alias, mb := range s.cfg.Mailboxes {
		c := client.New(mb.BaseURL, mb.JWT, mb.SitePassword)
		_, err := c.GetSettings()
		infos = append(infos, model.MailboxInfo{
			Alias:   alias,
			Name:    mb.Name,
			BaseURL: mb.BaseURL,
			Valid:   err == nil,
		})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Alias < infos[j].Alias })
	return mcp.NewToolResultText(toJSON(infos)), nil
}

func (s *Server) handleAddMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias, err := req.RequireString("alias")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: alias"), nil
	}
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: name"), nil
	}
	baseURL, err := req.RequireString("base_url")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: base_url"), nil
	}
	jwt, err := req.RequireString("jwt")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: jwt"), nil
	}
	sitePass := req.GetString("site_password", "")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cfg.Mailboxes[alias]; exists {
		return mcp.NewToolResultError(fmt.Sprintf("mailbox %q already exists, use remove_mailbox first or choose a different alias", alias)), nil
	}

	if s.cfg.DefaultMailbox == "" {
		s.cfg.DefaultMailbox = alias
	}
	s.cfg.Mailboxes[alias] = model.MailboxConfig{
		Name:         name,
		BaseURL:      baseURL,
		JWT:          jwt,
		SitePassword: sitePass,
	}
	if err := s.saveConfig(); err != nil {
		delete(s.cfg.Mailboxes, alias)
		if s.cfg.DefaultMailbox == alias {
			s.cfg.DefaultMailbox = ""
		}
		return mcp.NewToolResultError("save config: " + err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleRemoveMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias, err := req.RequireString("alias")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: alias"), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cfg.Mailboxes[alias]; !ok {
		return mcp.NewToolResultError(fmt.Sprintf("mailbox %q not found", alias)), nil
	}

	prevDefault := s.cfg.DefaultMailbox
	mbCopy := s.cfg.Mailboxes[alias]

	delete(s.cfg.Mailboxes, alias)
	if s.cfg.DefaultMailbox == alias {
		s.cfg.DefaultMailbox = ""
		keys := make([]string, 0, len(s.cfg.Mailboxes))
		for k := range s.cfg.Mailboxes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			s.cfg.DefaultMailbox = keys[0]
		}
	}
	if err := s.saveConfig(); err != nil {
		s.cfg.Mailboxes[alias] = mbCopy
		s.cfg.DefaultMailbox = prevDefault
		return mcp.NewToolResultError("save config: " + err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleSwitchMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias, err := req.RequireString("alias")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: alias"), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cfg.Mailboxes[alias]; !ok {
		return mcp.NewToolResultError(fmt.Sprintf("mailbox %q not found", alias)), nil
	}

	prevDefault := s.cfg.DefaultMailbox
	s.cfg.DefaultMailbox = alias
	if err := s.saveConfig(); err != nil {
		s.cfg.DefaultMailbox = prevDefault
		return mcp.NewToolResultError("save config: " + err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleValidateMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("alias", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	settings, err := c.GetSettings()
	if err != nil {
		return mcp.NewToolResultText(toJSON(map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})), nil
	}
	return mcp.NewToolResultText(toJSON(settings)), nil
}

func (s *Server) handleListEmails(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit, err := parseIntParam(req.GetString("limit", ""), "20", 1, 100)
	if err != nil {
		return mcp.NewToolResultError("invalid limit: " + err.Error()), nil
	}
	offset, err := parseIntParam(req.GetString("offset", ""), "0", 0, 10000)
	if err != nil {
		return mcp.NewToolResultError("invalid offset: " + err.Error()), nil
	}
	result, err := c.ListParsedMails(limit, offset)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *Server) handleGetEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := parseMailID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	mail, err := c.GetParsedMail(id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(mail)), nil
}

func (s *Server) handleDeleteEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := parseMailID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := c.DeleteMail(id); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleClearInbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := c.ClearInbox(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleSearchEmails(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: query"), nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return mcp.NewToolResultError("query must not be empty"), nil
	}
	limit, err := parseIntParam(req.GetString("limit", ""), "20", 1, 100)
	if err != nil {
		return mcp.NewToolResultError("invalid limit: " + err.Error()), nil
	}
	result, err := c.ListParsedMails(100, 0)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	q := strings.ToLower(query)
	filtered := make([]model.ParsedMail, 0)
	for _, m := range result.Results {
		if strings.Contains(strings.ToLower(m.Sender), q) || strings.Contains(strings.ToLower(m.Subject), q) {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return mcp.NewToolResultText(toJSON(map[string]interface{}{
		"results": filtered,
		"count":   len(filtered),
	})), nil
}

func (s *Server) handleSendEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	toMail, err := req.RequireString("to_mail")
	if err != nil || strings.TrimSpace(toMail) == "" {
		return mcp.NewToolResultError("missing or empty required parameter: to_mail"), nil
	}
	subject, err := req.RequireString("subject")
	if err != nil || strings.TrimSpace(subject) == "" {
		return mcp.NewToolResultError("missing or empty required parameter: subject"), nil
	}
	content, err := req.RequireString("content")
	if err != nil || strings.TrimSpace(content) == "" {
		return mcp.NewToolResultError("missing or empty required parameter: content"), nil
	}
	isHTML, err := strconv.ParseBool(req.GetString("is_html", "false"))
	if err != nil {
		return mcp.NewToolResultError("invalid is_html: must be true or false"), nil
	}

	body := &model.SendMailBody{
		ToMail:   toMail,
		ToName:   req.GetString("to_name", ""),
		FromName: req.GetString("from_name", ""),
		Subject:  subject,
		Content:  content,
		IsHTML:   isHTML,
	}
	if err := c.SendMail(body); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleCheckSendBalance(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	settings, err := c.GetSettings()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]int{"send_balance": settings.SendBalance})), nil
}

func (s *Server) handleListSent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit, err := parseIntParam(req.GetString("limit", ""), "20", 1, 100)
	if err != nil {
		return mcp.NewToolResultError("invalid limit: " + err.Error()), nil
	}
	offset, err := parseIntParam(req.GetString("offset", ""), "0", 0, 10000)
	if err != nil {
		return mcp.NewToolResultError("invalid offset: " + err.Error()), nil
	}
	result, err := c.ListSendbox(limit, offset)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *Server) handleDeleteSent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	idStr, err := req.RequireString("send_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: send_id"), nil
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return mcp.NewToolResultError("send_id must be a positive integer"), nil
	}
	if err := c.DeleteSendbox(id); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleClearSent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := c.ClearSentItems(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleGetAutoReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cfg, err := c.GetAutoReply()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(cfg)), nil
}

func (s *Server) handleSetAutoReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cfg := &model.AutoReplyConfig{}
	if v := req.GetString("subject", ""); v != "" {
		cfg.Subject = v
	}
	if v := req.GetString("message", ""); v != "" {
		cfg.Message = v
	}
	if v := req.GetString("enabled", ""); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return mcp.NewToolResultError("invalid enabled: must be true or false"), nil
		}
		cfg.Enabled = enabled
	}
	if v := req.GetString("name", ""); v != "" {
		cfg.Name = v
	}
	if err := c.SetAutoReply(cfg); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleGetWebhook(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cfg, err := c.GetWebhook()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(cfg)), nil
}

func (s *Server) handleSetWebhook(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	url, err := req.RequireString("url")
	if err != nil || strings.TrimSpace(url) == "" {
		return mcp.NewToolResultError("missing or empty required parameter: url"), nil
	}
	cfg := &model.WebhookSettings{URL: url}
	if err := c.SetWebhook(cfg); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleListAttachments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := c.ListAttachments()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(listMailboxesTool, s.handleListMailboxes)
	s.mcpServer.AddTool(addMailboxTool, s.handleAddMailbox)
	s.mcpServer.AddTool(removeMailboxTool, s.handleRemoveMailbox)
	s.mcpServer.AddTool(switchMailboxTool, s.handleSwitchMailbox)
	s.mcpServer.AddTool(validateMailboxTool, s.handleValidateMailbox)
	s.mcpServer.AddTool(listEmailsTool, s.handleListEmails)
	s.mcpServer.AddTool(getEmailTool, s.handleGetEmail)
	s.mcpServer.AddTool(deleteEmailTool, s.handleDeleteEmail)
	s.mcpServer.AddTool(clearInboxTool, s.handleClearInbox)
	s.mcpServer.AddTool(searchEmailsTool, s.handleSearchEmails)
	s.mcpServer.AddTool(sendEmailTool, s.handleSendEmail)
	s.mcpServer.AddTool(checkSendBalanceTool, s.handleCheckSendBalance)
	s.mcpServer.AddTool(listSentTool, s.handleListSent)
	s.mcpServer.AddTool(deleteSentTool, s.handleDeleteSent)
	s.mcpServer.AddTool(clearSentTool, s.handleClearSent)
	s.mcpServer.AddTool(getAutoReplyTool, s.handleGetAutoReply)
	s.mcpServer.AddTool(setAutoReplyTool, s.handleSetAutoReply)
	s.mcpServer.AddTool(getWebhookTool, s.handleGetWebhook)
	s.mcpServer.AddTool(setWebhookTool, s.handleSetWebhook)
	s.mcpServer.AddTool(listAttachmentsTool, s.handleListAttachments)
}

func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func parseMailID(req mcp.CallToolRequest) (int, error) {
	idStr, err := req.RequireString("mail_id")
	if err != nil {
		return 0, fmt.Errorf("missing required parameter: mail_id")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("mail_id must be a positive integer")
	}
	return id, nil
}

func parseIntParam(raw, defaultVal string, min, max int) (int, error) {
	if raw == "" {
		raw = defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}
	if v < min || v > max {
		return 0, fmt.Errorf("must be between %d and %d", min, max)
	}
	return v, nil
}

var listMailboxesTool = mcp.NewTool("list_mailboxes",
	mcp.WithDescription("List all configured mailboxes with their JWT validity status"),
)

var addMailboxTool = mcp.NewTool("add_mailbox",
	mcp.WithDescription("Add a new mailbox credential"),
	mcp.WithString("alias", mcp.Required(), mcp.Description("Short unique identifier for this mailbox")),
	mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable display name")),
	mcp.WithString("base_url", mcp.Required(), mcp.Description("API base URL of the cloudflare_temp_email instance")),
	mcp.WithString("jwt", mcp.Required(), mcp.Description("Address JWT credential from the web UI")),
	mcp.WithString("site_password", mcp.Description("Site-wide password if the deployment uses x-custom-auth")),
)

var removeMailboxTool = mcp.NewTool("remove_mailbox",
	mcp.WithDescription("Remove a mailbox and its credentials"),
	mcp.WithString("alias", mcp.Required(), mcp.Description("Alias of the mailbox to remove")),
)

var switchMailboxTool = mcp.NewTool("switch_mailbox",
	mcp.WithDescription("Set the default active mailbox"),
	mcp.WithString("alias", mcp.Required(), mcp.Description("Alias of the mailbox to make default")),
)

var validateMailboxTool = mcp.NewTool("validate_mailbox",
	mcp.WithDescription("Check if a mailbox JWT is still valid"),
	mcp.WithString("alias", mcp.Description("Mailbox alias to validate (leave empty for default)")),
)

var listEmailsTool = mcp.NewTool("list_emails",
	mcp.WithDescription("List received emails with pagination"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
	mcp.WithString("limit", mcp.Description("Number of emails to fetch (1-100, default 20)")),
	mcp.WithString("offset", mcp.Description("Offset for pagination (default 0)")),
)

var getEmailTool = mcp.NewTool("get_email",
	mcp.WithDescription("Get a single email with full parsed content"),
	mcp.WithString("mail_id", mcp.Required(), mcp.Description("Email ID to retrieve")),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var deleteEmailTool = mcp.NewTool("delete_email",
	mcp.WithDescription("Delete a single email"),
	mcp.WithString("mail_id", mcp.Required(), mcp.Description("Email ID to delete")),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var clearInboxTool = mcp.NewTool("clear_inbox",
	mcp.WithDescription("Delete all received emails in the inbox"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var searchEmailsTool = mcp.NewTool("search_emails",
	mcp.WithDescription("Search emails by sender or subject keyword (client-side search)"),
	mcp.WithString("query", mcp.Required(), mcp.Description("Keyword to search in sender and subject")),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
	mcp.WithString("limit", mcp.Description("Maximum results (default 20)")),
)

var sendEmailTool = mcp.NewTool("send_email",
	mcp.WithDescription("Send an email from the current mailbox"),
	mcp.WithString("to_mail", mcp.Required(), mcp.Description("Recipient email address")),
	mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
	mcp.WithString("content", mcp.Required(), mcp.Description("Email body content")),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
	mcp.WithString("from_name", mcp.Description("Display name for the sender")),
	mcp.WithString("to_name", mcp.Description("Display name for the recipient")),
	mcp.WithString("is_html", mcp.Description("Whether content is HTML (true/false, default false)")),
)

var checkSendBalanceTool = mcp.NewTool("check_send_balance",
	mcp.WithDescription("Check remaining send balance for the mailbox"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var listSentTool = mcp.NewTool("list_sent",
	mcp.WithDescription("List sent emails with pagination"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
	mcp.WithString("limit", mcp.Description("Number of items (1-100, default 20)")),
	mcp.WithString("offset", mcp.Description("Offset for pagination (default 0)")),
)

var deleteSentTool = mcp.NewTool("delete_sent",
	mcp.WithDescription("Delete a sent email record"),
	mcp.WithString("send_id", mcp.Required(), mcp.Description("ID of the sent email to delete")),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var clearSentTool = mcp.NewTool("clear_sent",
	mcp.WithDescription("Delete all sent email records"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var getAutoReplyTool = mcp.NewTool("get_auto_reply",
	mcp.WithDescription("Get the auto-reply configuration for the mailbox"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var setAutoReplyTool = mcp.NewTool("set_auto_reply",
	mcp.WithDescription("Configure auto-reply settings"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
	mcp.WithString("subject", mcp.Description("Auto-reply subject")),
	mcp.WithString("message", mcp.Description("Auto-reply message body")),
	mcp.WithString("enabled", mcp.Description("Enable auto-reply (true/false)")),
	mcp.WithString("name", mcp.Description("Auto-reply sender name")),
)

var getWebhookTool = mcp.NewTool("get_webhook",
	mcp.WithDescription("Get the webhook configuration for the mailbox"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)

var setWebhookTool = mcp.NewTool("set_webhook",
	mcp.WithDescription("Configure webhook settings"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
	mcp.WithString("url", mcp.Required(), mcp.Description("Webhook URL to receive notifications")),
)

var listAttachmentsTool = mcp.NewTool("list_attachments",
	mcp.WithDescription("List S3 attachments for the mailbox"),
	mcp.WithString("mailbox", mcp.Description("Mailbox alias (default if empty)")),
)
