package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"agent-mail/mcp"
	_ "agent-mail/provider/cloudflare"
	_ "agent-mail/provider/gmail"
	_ "agent-mail/provider/qqmail"
	"agent-mail/service"
	"agent-mail/store/sqlite"
	"agent-mail/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address for HTTP MCP server")
	dbPath := flag.String("db-path", "", "path to SQLite database file (default: $HOME/.agent-mail/data.db)")
	envFile := flag.String("env-file", "", "path to .env file")
	authHeader := flag.String("auth-header", os.Getenv("AUTH_HEADER"), "HTTP header name for auth (env: AUTH_HEADER)")
	authToken := flag.String("auth-token", os.Getenv("AUTH_TOKEN"), "auth token value (env: AUTH_TOKEN)")
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

	if *authHeader != "" {
		os.Setenv("AUTH_HEADER", *authHeader)
	}
	if *authToken != "" {
		os.Setenv("AUTH_TOKEN", *authToken)
	}

	db, err := sqlite.Open(path)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database opened", "path", path)

	initAdminPassword(db)

	mailboxSvc := service.NewMailboxService(db, nil)
	emailSvc := service.NewEmailService(mailboxSvc)
	sendSvc := service.NewSendService(mailboxSvc)
	autoReplySvc := service.NewAutoReplyService(mailboxSvc)
	webhookSvc := service.NewWebhookService(mailboxSvc)
	attachmentSvc := service.NewAttachmentService(mailboxSvc)
	userSvc := service.NewUserService(db)

	handler := mcp.NewHandler(mailboxSvc, emailSvc, sendSvc, autoReplySvc, webhookSvc, attachmentSvc)
	srv := mcp.NewServer(*addr, handler, userSvc)

	adminHandler := web.NewAdminHandler(db, userSvc)
	srv.RegisterAdmin(adminHandler.Register)

	userHandler := web.NewUserHandler(userSvc, mailboxSvc)
	srv.RegisterAdmin(userHandler.Register)

	slog.Info("agent-mail starting", "addr", *addr)
	if err := srv.Start(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func initAdminPassword(db *sqlite.DB) {
	stored, err := db.GetSetting("admin_password_hash")
	if err != nil {
		slog.Error("failed to read admin password", "error", err)
		os.Exit(1)
	}
	if stored != "" {
		return
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		b := make([]byte, 16)
		rand.Read(b)
		password = hex.EncodeToString(b)
		slog.Warn("ADMIN_PASSWORD not set, generated random password", "password", password)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash admin password", "error", err)
		os.Exit(1)
	}
	db.SetSetting("admin_password_hash", string(hash))
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
