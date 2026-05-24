package mcp

import (
	"encoding/json"
	"fmt"

	"agent-mail/client"
	"agent-mail/config"
	"agent-mail/model"

	"github.com/mark3labs/mcp-go/server"
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

func (s *Server) registerTools() {
	// Tools are registered in Tasks 2-5
}

func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
