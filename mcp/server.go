package mcp

import (
	"context"
	"log/slog"
	"net/http"

	mcp "github.com/mark3labs/mcp-go/mcp"
	goserver "github.com/mark3labs/mcp-go/server"
)

type Server struct {
	httpServer *http.Server
	handler    *Handler
}

func NewServer(addr string, handler *Handler) *Server {
	return &Server{
		httpServer: &http.Server{Addr: addr},
		handler:    handler,
	}
}

func (s *Server) Start() error {
	mcpSrv := goserver.NewMCPServer("agent-mail", "1.0.0")

	for _, tool := range Tools {
		mcpSrv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return s.handler.HandleToolCall(ctx, req)
		})
	}

	streamableServer := goserver.NewStreamableHTTPServer(mcpSrv)

	mux := http.NewServeMux()
	mux.Handle("/mcp", streamableServer)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	s.httpServer.Handler = mux
	slog.Info("MCP server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}
