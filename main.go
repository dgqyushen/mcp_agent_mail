package main

import (
	"flag"
	"fmt"
	"os"

	"agent-mail/config"
	mcp "agent-mail/mcp"
	"agent-mail/model"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: ~/.agent-mail/config.json)")
	transport := flag.String("transport", "stdio", "transport mode: stdio, sse, streamable-http")
	addr := flag.String("addr", ":8080", "listen address for HTTP based transport")
	flag.Parse()

	path := *cfgPath
	if path == "" {
		path = model.DefaultConfigPath()
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	s := mcp.New(cfg, path)

	switch *transport {
	case "sse":
		sseServer := server.NewSSEServer(s.MCPServer())
		fmt.Fprintf(os.Stderr, "Starting SSE MCP server on %s\n", *addr)
		if err := sseServer.Start(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	case "streamable-http":
		httpServer := server.NewStreamableHTTPServer(s.MCPServer())
		fmt.Fprintf(os.Stderr, "Starting Streamable HTTP MCP server on %s\n", *addr)
		if err := httpServer.Start(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	default:
		if err := s.ServeStdio(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}
}
