# Agent-Mail MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local MCP server that wraps cloudflare_temp_email API for agents (Hermes/OpenClaw) — multi-mailbox, full CRUD.

**Architecture:** Go binary, MCP stdio transport, config at `~/.agent-mail/config.json`, live queries to cloudflare_temp_email API (no local storage).

**Tech Stack:** Go 1.22, `github.com/mark3labs/mcp-go`, `net/http`, `encoding/json`, `FROM scratch` Docker.

---

### Task 1: Initialize Go Module and Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `model/types.go`
- Create: `config/config.go`
- Create: `client/api.go`
- Create: `mcp/tools.go`
- Create: `main.go`

- [ ] **Step 1: Initialize go module**

Run: `go mod init agent-mail`
Expected: `created go.mod`

Write `go.mod`:
```
module agent-mail

go 1.22
```

- [ ] **Step 2: Install mcp-go dependency**

Run: `go get github.com/mark3labs/mcp-go@latest`
Expected: go.sum created, mod updated

Run: `go mod tidy`
Expected: clean output

- [ ] **Step 3: Create placeholder files**

```bash
mkdir -p model config client mcp
touch model/types.go config/config.go client/api.go mcp/tools.go main.go
```

- [ ] **Step 4: Write model/types.go**

```go
package model

import (
	"os"
	"path/filepath"
	"time"
)

type MailboxConfig struct {
	Name         string `json:"name"`
	BaseURL      string `json:"base_url"`
	JWT          string `json:"jwt"`
	SitePassword string `json:"site_password"`
}

type Config struct {
	DefaultMailbox string                    `json:"default_mailbox"`
	Mailboxes      map[string]MailboxConfig  `json:"mailboxes"`
}

type SettingsResponse struct {
	Address     string `json:"address"`
	SendBalance int    `json:"send_balance"`
}

type ParsedMail struct {
	ID        int            `json:"id"`
	MessageID string         `json:"message_id"`
	Source    string         `json:"source"`
	To        string         `json:"to"`
	CreatedAt string         `json:"created_at"`
	Sender    string         `json:"sender"`
	Subject   string         `json:"subject"`
	Text      string         `json:"text"`
	HTML      string         `json:"html"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	MimeType    string `json:"mimeType"`
	Disposition string `json:"disposition"`
	Size        int    `json:"size"`
}

type MailboxInfo struct {
	Alias   string `json:"alias"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Valid   bool   `json:"valid"`
}

type SendMailBody struct {
	FromName string `json:"from_name"`
	ToMail   string `json:"to_mail"`
	ToName   string `json:"to_name"`
	Subject  string `json:"subject"`
	Content  string `json:"content"`
	IsHTML   bool   `json:"is_html"`
}

type AutoReplyConfig struct {
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	SourcePrefix string `json:"source_prefix"`
	Message      string `json:"message"`
	Enabled      bool   `json:"enabled"`
}

type WebhookSettings struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type PaginatedResult struct {
	Results []ParsedMail `json:"results"`
	Count   int          `json:"count"`
}

type SendboxItem struct {
	ID        int    `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"created_at"`
}

type SendboxResult struct {
	Results []SendboxItem `json:"results"`
	Count   int           `json:"count"`
}

type AttachmentItem struct {
	Key string `json:"key"`
}

type AttachmentListResult struct {
	Results []AttachmentItem `json:"results"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type SentEmailSummary struct {
	ID        int    `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"created_at"`
}

type ConfigPathFunc func() string
var DefaultConfigPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agent-mail", "config.json")
}
```

- [ ] **Step 5: Write config/config.go**

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"agent-mail/model"
)

func Load(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.Config{
				DefaultMailbox: "",
				Mailboxes:      make(map[string]model.MailboxConfig),
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Mailboxes == nil {
		cfg.Mailboxes = make(map[string]model.MailboxConfig)
	}
	return &cfg, nil
}

func Save(path string, cfg *model.Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
```

- [ ] **Step 6: Write initial client/api.go**

```go
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-mail/model"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	jwt        string
	sitePass   string
}

func New(baseURL, jwt, sitePass string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		jwt:        jwt,
		sitePass:   sitePass,
	}
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	req.Header.Set("x-lang", "en")
	if c.sitePass != "" {
		req.Header.Set("x-custom-auth", c.sitePass)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	if resp.StatusCode == 429 {
		time.Sleep(3 * time.Second)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("retry %s %s: %w", method, path, err)
		}
	}
	return resp, nil
}

func (c *Client) GetSettings() (*model.SettingsResponse, error) {
	resp, err := c.doRequest("GET", "/api/settings", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get settings: %s %s", resp.Status, string(body))
	}
	var result model.SettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListParsedMails(limit, offset int) (*model.PaginatedResult, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/parsed_mails?limit=%d&offset=%d", limit, offset), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list mails: %s %s", resp.Status, string(body))
	}
	var result model.PaginatedResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetParsedMail(id int) (*model.ParsedMail, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/parsed_mail/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get mail %d: %s %s", id, resp.Status, string(body))
	}
	var result model.ParsedMail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteMail(id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/mails/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete mail %d: %s %s", id, resp.Status, string(body))
	}
	return nil
}

func (c *Client) ClearInbox() error {
	resp, err := c.doRequest("DELETE", "/api/clear_inbox", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clear inbox: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) SendMail(body *model.SendMailBody) error {
	data, _ := json.Marshal(body)
	resp, err := c.doRequest("POST", "/api/send_mail", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send mail: %s %s", resp.Status, string(respBody))
	}
	return nil
}

func (c *Client) ListSendbox(limit, offset int) (*model.SendboxResult, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/sendbox?limit=%d&offset=%d", limit, offset), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list sendbox: %s %s", resp.Status, string(body))
	}
	var result model.SendboxResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteSendbox(id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/sendbox/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete sent %d: %s %s", id, resp.Status, string(body))
	}
	return nil
}

func (c *Client) ClearSentItems() error {
	resp, err := c.doRequest("DELETE", "/api/clear_sent_items", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clear sent: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) GetAutoReply() (*model.AutoReplyConfig, error) {
	resp, err := c.doRequest("GET", "/api/auto_reply", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get auto reply: %s %s", resp.Status, string(body))
	}
	var result model.AutoReplyConfig
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SetAutoReply(cfg *model.AutoReplyConfig) error {
	data, _ := json.Marshal(map[string]*model.AutoReplyConfig{"auto_reply": cfg})
	resp, err := c.doRequest("POST", "/api/auto_reply", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set auto reply: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) GetWebhook() (*model.WebhookSettings, error) {
	resp, err := c.doRequest("GET", "/api/webhook/settings", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get webhook: %s %s", resp.Status, string(body))
	}
	var result model.WebhookSettings
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SetWebhook(cfg *model.WebhookSettings) error {
	data, _ := json.Marshal(cfg)
	resp, err := c.doRequest("POST", "/api/webhook/settings", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set webhook: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) ListAttachments() (*model.AttachmentListResult, error) {
	resp, err := c.doRequest("GET", "/api/attachment/list", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list attachments: %s %s", resp.Status, string(body))
	}
	var result model.AttachmentListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
```

- [ ] **Step 7: Write initial mcp/tools.go as placeholder**

```go
package mcp

import (
	"encoding/json"
	"fmt"

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
```

- [ ] **Step 8: Write initial main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"agent-mail/config"
	mcp "agent-mail/mcp"
	"agent-mail/model"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: ~/.agent-mail/config.json)")
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
	if err := s.ServeStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 9: Verify build compiles**

Run: `GOOS=linux CGO_ENABLED=0 go build -o /dev/null .`
Expected: exits 0, no output

- [ ] **Step 10: Commit**

```bash
git add -A && git commit -m "feat: scaffold Go module with model, config, client, tools skeleton"
```

---

### Task 2: Write MCP Tool Definitions (mailbox management)

**Files:**
- Modify: `mcp/tools.go`

This step adds the 5 mailbox management tool definitions and their handlers.

- [ ] **Step 1: Add mailbox management tool definitions and handlers above `registerTools`**

Add after `package mcp`:

```go
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
```

- [ ] **Step 2: Add handlers**

```go
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
```

- [ ] **Step 3: Update registerTools to include mailbox tools**

Replace the empty `registerTools()` body with:

```go
func (s *Server) registerTools() {
	s.mcpServer.AddTool(listMailboxesTool, s.handleListMailboxes)
	s.mcpServer.AddTool(addMailboxTool, s.handleAddMailbox)
	s.mcpServer.AddTool(removeMailboxTool, s.handleRemoveMailbox)
	s.mcpServer.AddTool(switchMailboxTool, s.handleSwitchMailbox)
	s.mcpServer.AddTool(validateMailboxTool, s.handleValidateMailbox)
}
```

- [ ] **Step 4: Verify build**

Run: `GOOS=linux CGO_ENABLED=0 go build -o /dev/null .`
Expected: exits 0

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: add mailbox management MCP tools (5 tools)"
```

---

### Task 3: Write MCP Tool Definitions (email read + search)

**Files:**
- Modify: `mcp/tools.go`

- [ ] **Step 1: Add email read tool definitions**

```go
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
```

- [ ] **Step 2: Add handlers**

```go
func (s *Server) handleListEmails(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := 20
	if v := req.GetString("limit", ""); v != "" {
		limit, _ = strconv.Atoi(v)
	}
	offset := 0
	if v := req.GetString("offset", ""); v != "" {
		offset, _ = strconv.Atoi(v)
	}
	result, err := c.ListParsedMails(limit, offset)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *Server) handleGetEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	idStr, _ := req.RequireString("mail_id")
	id, _ := strconv.Atoi(idStr)
	mail, err := c.GetParsedMail(id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(mail)), nil
}

func (s *Server) handleDeleteEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	idStr, _ := req.RequireString("mail_id")
	id, _ := strconv.Atoi(idStr)
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
	query, _ := req.RequireString("query")
	limit := 20
	if v := req.GetString("limit", ""); v != "" {
		limit, _ = strconv.Atoi(v)
	}
	result, err := c.ListParsedMails(100, 0)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	q := strings.ToLower(query)
	var filtered []model.ParsedMail
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
```

- [ ] **Step 3: Update registerTools to include email tools**

```go
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
}
```

- [ ] **Step 4: Verify build**

Run: `GOOS=linux CGO_ENABLED=0 go build -o /dev/null .`
Expected: exits 0

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: add email read, delete, clear, search MCP tools (5 tools)"
```

---

### Task 4: Write MCP Tool Definitions (email send + sendbox)

**Files:**
- Modify: `mcp/tools.go`

- [ ] **Step 1: Add send/sendbox tool definitions**

```go
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
```

- [ ] **Step 2: Add handlers**

```go
func (s *Server) handleSendEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	c, err := s.getClientForMailbox(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	toMail, _ := req.RequireString("to_mail")
	subject, _ := req.RequireString("subject")
	content, _ := req.RequireString("content")
	isHTML := req.GetString("is_html", "false") == "true"

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
	limit := 20
	if v := req.GetString("limit", ""); v != "" {
		limit, _ = strconv.Atoi(v)
	}
	offset := 0
	if v := req.GetString("offset", ""); v != "" {
		offset, _ = strconv.Atoi(v)
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
	idStr, _ := req.RequireString("send_id")
	id, _ := strconv.Atoi(idStr)
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
```

- [ ] **Step 3: Update registerTools to include send/sendbox tools**

```go
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
}
```

- [ ] **Step 4: Verify build**

Run: `GOOS=linux CGO_ENABLED=0 go build -o /dev/null .`
Expected: exits 0

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: add email send and sendbox MCP tools (5 tools)"
```

---

### Task 5: Write MCP Tool Definitions (advanced: auto-reply, webhook, attachments)

**Files:**
- Modify: `mcp/tools.go`

- [ ] **Step 1: Add advanced tool definitions**

```go
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
```

- [ ] **Step 2: Add handlers**

```go
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
		cfg.Enabled = v == "true"
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
	url, _ := req.RequireString("url")
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
```

- [ ] **Step 3: Update registerTools to include all tools**

```go
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
```

- [ ] **Step 4: Verify build**

Run: `GOOS=linux CGO_ENABLED=0 go build -o /dev/null .`
Expected: exits 0

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: add auto-reply, webhook, attachment MCP tools (5 tools)"
```

---

### Task 6: Write Unit Tests

**Files:**
- Create: `config/config_test.go`
- Create: `client/api_test.go`
- Create: `mcp/tools_test.go`

- [ ] **Step 1: Write config/config_test.go**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"agent-mail/model"
)

func TestLoad_FileNotExists(t *testing.T) {
	cfg, err := Load("/tmp/nonexistent/config.json")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if cfg.DefaultMailbox != "" {
		t.Fatalf("expected empty default, got %q", cfg.DefaultMailbox)
	}
	if len(cfg.Mailboxes) != 0 {
		t.Fatalf("expected empty mailboxes, got %d", len(cfg.Mailboxes))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := &model.Config{
		DefaultMailbox: "work",
		Mailboxes: map[string]model.MailboxConfig{
			"work": {Name: "Work", BaseURL: "https://mail.example.com", JWT: "jwt123"},
		},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.DefaultMailbox != "work" {
		t.Fatalf("expected default=work, got %q", loaded.DefaultMailbox)
	}
	mb, ok := loaded.Mailboxes["work"]
	if !ok {
		t.Fatal("expected work mailbox")
	}
	if mb.Name != "Work" || mb.BaseURL != "https://mail.example.com" || mb.JWT != "jwt123" {
		t.Fatalf("unexpected mailbox: %+v", mb)
	}
}

func TestLoad_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("{bad json}"), 0600)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
```

- [ ] **Step 2: Write mcp/tools_test.go**

```go
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agent-mail/model"

	"github.com/mark3labs/mcp-go/mcp"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := &model.Config{
		DefaultMailbox: "test",
		Mailboxes: map[string]model.MailboxConfig{
			"test": {Name: "Test", BaseURL: "http://placeholder", JWT: "test-jwt"},
		},
	}
	s := New(cfg, path)
	return s, path
}

func TestListMailboxes(t *testing.T) {
	s, _ := newTestServer(t)
	result, err := s.handleListMailboxes(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected no error")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./config/ -v`
Expected: 3/3 tests pass

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "test: add config unit tests and mcp handler test scaffold"
```

---

### Task 7: Dockerfile and docker-compose

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Modify: `go.mod` (ensure build constraints)

- [ ] **Step 1: Write Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o agent-mail .

FROM scratch
COPY --from=builder /build/agent-mail /agent-mail
ENTRYPOINT ["/agent-mail"]
```

- [ ] **Step 2: Write docker-compose.yml**

```yaml
services:
  agent-mail:
    build: .
    volumes:
      - ~/.agent-mail:/root/.agent-mail
    restart: unless-stopped
```

- [ ] **Step 3: Build Docker image**

Run: `docker build -t agent-mail .`
Expected: builds successfully, image < 15MB

- [ ] **Step 4: Verify image size**

Run: `docker images agent-mail --format "{{.Size}}"`
Expected: ~10-15MB

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: add Dockerfile and docker-compose for deployment"
```

---

### Task 8: Final Verification and .gitignore

**Files:**
- Create: `.gitignore`
- Modify: `main.go` (final check)

- [ ] **Step 1: Write .gitignore**

```
agent-mail
*.test
```

- [ ] **Step 2: Full build and test**

```bash
CGO_ENABLED=0 go build -o agent-mail .
go test ./... -v
```

Expected: build succeeds, all tests pass

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "chore: add .gitignore, finalize"
```

---

## Spec Coverage Check

| Spec Requirement | Task |
|---|---|
| Multi-mailbox management (list/add/remove/switch/validate) | Task 2 |
| List emails with pagination | Task 3 |
| Get single email | Task 3 |
| Delete email | Task 3 |
| Clear inbox | Task 3 |
| Search emails (client-side) | Task 3 |
| Send email | Task 4 |
| Send balance check | Task 4 |
| List sent | Task 4 |
| Delete sent | Task 4 |
| Clear sent | Task 4 |
| Auto-reply get/set | Task 5 |
| Webhook get/set | Task 5 |
| List attachments | Task 5 |
| Config file persistence | Task 1 |
| Docker deployment | Task 7 |
| Unit tests | Task 6 |
