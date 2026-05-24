package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"

	"agent-mail/mcp"
	"agent-mail/service"
	"agent-mail/store/sqlite"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address for HTTP MCP server")
	dbPath := flag.String("db-path", "", "path to SQLite database file (default: $HOME/.agent-mail/data.db)")
	envFile := flag.String("env-file", "", "path to .env file")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	path := *dbPath
	if path == "" {
		home, _ := os.UserHomeDir()
		path = home + "/.agent-mail/data.db"
	}

	if *envFile != "" {
		loadEnvFile(*envFile)
	}

	db, err := sqlite.Open(path)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database opened", "path", path)

	mailboxSvc := service.NewMailboxService(db, nil)
	emailSvc := service.NewEmailService(mailboxSvc)
	sendSvc := service.NewSendService(mailboxSvc)
	autoReplySvc := service.NewAutoReplyService(mailboxSvc)
	webhookSvc := service.NewWebhookService(mailboxSvc)
	attachmentSvc := service.NewAttachmentService(mailboxSvc)

	handler := mcp.NewHandler(mailboxSvc, emailSvc, sendSvc, autoReplySvc, webhookSvc, attachmentSvc)
	srv := mcp.NewServer(*addr, handler)

	slog.Info("agent-mail starting", "addr", *addr)
	if err := srv.Start(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("failed to read env file", "path", path, "error", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
