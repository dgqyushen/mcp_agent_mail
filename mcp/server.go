package mcp

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	mcp "github.com/mark3labs/mcp-go/mcp"
	goserver "github.com/mark3labs/mcp-go/server"

	"agent-mail/service"
)

type Server struct {
	httpServer   *http.Server
	handler      *Handler
	userSvc      *service.UserService
	mux          *http.ServeMux
	adminHandler func(mux *http.ServeMux)
}

func (s *Server) RegisterAdmin(fn func(mux *http.ServeMux)) {
	s.adminHandler = fn
}

func NewServer(addr string, handler *Handler, userSvc *service.UserService) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		handler: handler,
		userSvc: userSvc,
	}
}

func (s *Server) ServeMux() *http.ServeMux { return s.mux }

func (s *Server) Start() error {
	mcpSrv := goserver.NewMCPServer("agent-mail", "1.0.0")

	for _, tool := range Tools {
		mcpSrv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return s.handler.HandleToolCall(ctx, req)
		})
	}

	streamableServer := goserver.NewStreamableHTTPServer(mcpSrv)

	mux := http.NewServeMux()
	mux.Handle("/mcp", AuthMiddleware(streamableServer, s.userSvc))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	if s.adminHandler != nil {
		s.adminHandler(mux)
	}
	s.mux = mux
	s.httpServer.Handler = mux
	slog.Info("MCP server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}
