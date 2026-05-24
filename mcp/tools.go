package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"agent-mail/client"
	"agent-mail/config"
	"agent-mail/model"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/mcp"
)

type Server struct {
	mcpServer *server.MCPServer
	cfg       *model.Config
	cfgPath   string
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
	alias, _ := req.RequireString("alias")
	name, _ := req.RequireString("name")
	baseURL, _ := req.RequireString("base_url")
	jwt, _ := req.RequireString("jwt")
	sitePass := req.GetString("site_password", "")

	s.cfg.Mailboxes[alias] = model.MailboxConfig{
		Name:         name,
		BaseURL:      baseURL,
		JWT:          jwt,
		SitePassword: sitePass,
	}
	if s.cfg.DefaultMailbox == "" {
		s.cfg.DefaultMailbox = alias
	}
	if err := s.saveConfig(); err != nil {
		return mcp.NewToolResultError("save config: " + err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleRemoveMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias, _ := req.RequireString("alias")
	if _, ok := s.cfg.Mailboxes[alias]; !ok {
		return mcp.NewToolResultError(fmt.Sprintf("mailbox %q not found", alias)), nil
	}
	delete(s.cfg.Mailboxes, alias)
	if s.cfg.DefaultMailbox == alias {
		s.cfg.DefaultMailbox = ""
		for k := range s.cfg.Mailboxes {
			s.cfg.DefaultMailbox = k
			break
		}
	}
	if err := s.saveConfig(); err != nil {
		return mcp.NewToolResultError("save config: " + err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleSwitchMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias, _ := req.RequireString("alias")
	if _, ok := s.cfg.Mailboxes[alias]; !ok {
		return mcp.NewToolResultError(fmt.Sprintf("mailbox %q not found", alias)), nil
	}
	s.cfg.DefaultMailbox = alias
	if err := s.saveConfig(); err != nil {
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
		return mcp.NewToolResultError("JWT invalid or expired: " + err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(settings)), nil
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(listMailboxesTool, s.handleListMailboxes)
	s.mcpServer.AddTool(addMailboxTool, s.handleAddMailbox)
	s.mcpServer.AddTool(removeMailboxTool, s.handleRemoveMailbox)
	s.mcpServer.AddTool(switchMailboxTool, s.handleSwitchMailbox)
	s.mcpServer.AddTool(validateMailboxTool, s.handleValidateMailbox)
}

func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
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
