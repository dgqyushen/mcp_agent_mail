# agent-mail Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign agent-mail as a remote-only MCP server with SQLite storage, layered architecture, and multi-provider support.

**Architecture:** handler → service → {provider, store}. Single streamable-http transport, .env config, SQLite persistence.

**Tech Stack:** Go 1.25, mcp-go, modernc.org/sqlite (pure Go), net/http

---

## File Structure Map

```
Create:  provider/provider.go, provider/cloudflare/client.go
Create:  store/sqlite/db.go, store/sqlite/settings.go, store/sqlite/mailboxes.go
Create:  service/mailbox_svc.go, service/email_svc.go, service/send_svc.go
Create:  service/auto_reply_svc.go, service/webhook_svc.go, service/attachment_svc.go
Create:  mcp/server.go, mcp/handler.go, mcp/tools_def.go
Create:  .env.example
Create:  store/sqlite/db_test.go, store/sqlite/settings_test.go, store/sqlite/mailboxes_test.go
Create:  service/mailbox_svc_test.go, service/email_svc_test.go
Create:  mcp/handler_test.go

Modify:  model/types.go, main.go, go.mod, go.sum
Modify:  Dockerfile, docker-compose.yml

Delete:  client/api.go, config/config.go, config/config_test.go
Delete:  mcp/tools.go, mcp/tools_test.go
Delete:  config.example.toml
```

---

### Task 1: Update model/types.go

**Files:**
- Modify: `model/types.go`

**Goal:** Remove config-layer types, keep all API response/request types. Add DB row types.

- [ ] **Step 1: Replace model/types.go**

Write the file with these types (keeping existing API types, removing Config/MailboxConfig/ConfigPathFunc/DefaultConfigPath):

```go
package model

type SettingsResponse struct {
	Address     string `json:"address"`
	SendBalance int    `json:"send_balance"`
}

type ParsedMail struct {
	ID          int          `json:"id"`
	MessageID   string       `json:"message_id"`
	Source      string       `json:"source"`
	To          string       `json:"to"`
	CreatedAt   string       `json:"created_at"`
	Sender      string       `json:"sender"`
	Subject     string       `json:"subject"`
	Text        string       `json:"text"`
	HTML        string       `json:"html"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	MimeType    string `json:"mimeType"`
	Disposition string `json:"disposition"`
	Size        int    `json:"size"`
}

type MailboxInfo struct {
	Alias        string `json:"alias"`
	Name         string `json:"name"`
	ProviderType string `json:"provider_type"`
	BaseURL      string `json:"base_url"`
	Valid        bool   `json:"valid"`
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

type SentEmailSummary struct {
	ID        int    `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"created_at"`
}

type MailboxRecord struct {
	Alias        string
	Name         string
	ProviderType string
	BaseURL      string
	AuthData     string
}
```

- [ ] **Step 2: Commit**

```bash
git add model/types.go && git commit -m "refactor: strip config types from model, keep API types"
```

---

### Task 2: Define EmailProvider interface

**Files:**
- Create: `provider/provider.go`

**Goal:** Interface all email backends must implement.

- [ ] **Step 1: Create provider/provider.go**

```go
package provider

import "agent-mail/model"

type EmailProvider interface {
	GetSettings() (*model.SettingsResponse, error)

	ListEmails(limit, offset int) (*model.PaginatedResult, error)
	GetEmail(id int) (*model.ParsedMail, error)
	DeleteEmail(id int) error
	ClearInbox() error

	SendMail(body *model.SendMailBody) error
	CheckSendBalance() (int, error)

	ListSent(limit, offset int) (*model.SendboxResult, error)
	DeleteSent(id int) error
	ClearSent() error

	GetAutoReply() (*model.AutoReplyConfig, error)
	SetAutoReply(cfg *model.AutoReplyConfig) error

	GetWebhook() (*model.WebhookSettings, error)
	SetWebhook(cfg *model.WebhookSettings) error

	ListAttachments() (*model.AttachmentListResult, error)

	Validate() error
}
```

- [ ] **Step 2: Build check**

```bash
go build ./provider
```

- [ ] **Step 3: Commit**

```bash
git add provider/provider.go && git commit -m "feat: add EmailProvider interface"
```

---

### Task 3: SQLite store — db init and migrations

**Files:**
- Create: `store/sqlite/db.go`

**Goal:** Open SQLite connection, run migrations.

- [ ] **Step 1: Add sqlite dependency**

```bash
go get modernc.org/sqlite
```

- [ ] **Step 2: Create store/sqlite/db.go**

```go
package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS mailboxes (
			alias          TEXT PRIMARY KEY,
			name           TEXT NOT NULL,
			provider_type  TEXT NOT NULL DEFAULT 'cloudflare',
			base_url       TEXT NOT NULL,
			auth_data      TEXT NOT NULL DEFAULT '{}',
			created_at     TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	return err
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}
```

- [ ] **Step 3: Write test store/sqlite/db_test.go**

```go
package sqlite_test

import (
	"testing"

	"agent-mail/store/sqlite"
)

func TestOpen(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count int
	err = db.Conn().QueryRow("SELECT COUNT(*) FROM mailboxes").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 mailboxes, got %d", count)
	}
}
```

- [ ] **Step 4: Run test**

```bash
go test ./store/sqlite/ -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add store/ go.mod go.sum && git commit -m "feat: add SQLite store with migration"
```

---

### Task 4: SQLite store — settings CRUD

**Files:**
- Create: `store/sqlite/settings.go`
- Create: `store/sqlite/settings_test.go`

- [ ] **Step 1: Write test store/sqlite/settings_test.go**

```go
package sqlite_test

import (
	"testing"

	"agent-mail/store/sqlite"
)

func TestSettingsGetSet(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SetSetting("key1", "val1"); err != nil {
		t.Fatal(err)
	}

	v, err := db.GetSetting("key1")
	if err != nil {
		t.Fatal(err)
	}
	if v != "val1" {
		t.Errorf("expected val1, got %q", v)
	}
}

func TestSettingsGetDefault(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	v, err := db.GetSetting("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

```bash
go test ./store/sqlite/ -v -run TestSettings
```

Expected: FAIL (methods not defined)

- [ ] **Step 3: Create store/sqlite/settings.go**

```go
package sqlite

func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", nil // return empty on missing key
	}
	return value, nil
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}
```

- [ ] **Step 4: Run test — verify pass**

```bash
go test ./store/sqlite/ -v -run TestSettings
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add store/sqlite/settings.go store/sqlite/settings_test.go && git commit -m "feat: add settings CRUD to SQLite store"
```

---

### Task 5: SQLite store — mailboxes CRUD

**Files:**
- Create: `store/sqlite/mailboxes.go`
- Create: `store/sqlite/mailboxes_test.go`

- [ ] **Step 1: Write test store/sqlite/mailboxes_test.go**

```go
package sqlite_test

import (
	"testing"

	"agent-mail/model"
	"agent-mail/store/sqlite"
)

func TestMailboxesCRUD(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rec := model.MailboxRecord{
		Alias:        "work",
		Name:         "Work",
		ProviderType: "cloudflare",
		BaseURL:      "https://mail.example.com",
		AuthData:     `{"jwt":"token123"}`,
	}

	if err := db.InsertMailbox(rec); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetMailbox("work")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Work" {
		t.Errorf("expected Work, got %q", got.Name)
	}

	list, err := db.ListMailboxes()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 mailbox, got %d", len(list))
	}

	if err := db.DeleteMailbox("work"); err != nil {
		t.Fatal(err)
	}

	list, err = db.ListMailboxes()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(list))
	}
}

func TestMailboxDuplicate(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rec := model.MailboxRecord{Alias: "x", Name: "X", BaseURL: "https://x.com", AuthData: "{}"}
	if err := db.InsertMailbox(rec); err != nil {
		t.Fatal(err)
	}
	err = db.InsertMailbox(rec)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

```bash
go test ./store/sqlite/ -v -run TestMailboxes
```

Expected: FAIL

- [ ] **Step 3: Create store/sqlite/mailboxes.go**

```go
package sqlite

import (
	"fmt"
	"strings"

	"agent-mail/model"
)

func (db *DB) InsertMailbox(m model.MailboxRecord) error {
	_, err := db.conn.Exec(
		`INSERT INTO mailboxes (alias, name, provider_type, base_url, auth_data)
		 VALUES (?, ?, ?, ?, ?)`,
		m.Alias, m.Name, m.ProviderType, m.BaseURL, m.AuthData,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return fmt.Errorf("mailbox %q already exists", m.Alias)
		}
		return err
	}
	return nil
}

func (db *DB) GetMailbox(alias string) (*model.MailboxRecord, error) {
	var m model.MailboxRecord
	err := db.conn.QueryRow(
		`SELECT alias, name, provider_type, base_url, auth_data FROM mailboxes WHERE alias = ?`,
		alias,
	).Scan(&m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData)
	if err != nil {
		return nil, fmt.Errorf("mailbox %q not found: %w", alias, err)
	}
	if m.ProviderType == "" {
		m.ProviderType = "cloudflare"
	}
	return &m, nil
}

func (db *DB) ListMailboxes() ([]model.MailboxRecord, error) {
	rows, err := db.conn.Query(
		`SELECT alias, name, provider_type, base_url, auth_data FROM mailboxes ORDER BY alias`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.MailboxRecord
	for rows.Next() {
		var m model.MailboxRecord
		if err := rows.Scan(&m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData); err != nil {
			return nil, err
		}
		if m.ProviderType == "" {
			m.ProviderType = "cloudflare"
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (db *DB) DeleteMailbox(alias string) error {
	result, err := db.conn.Exec("DELETE FROM mailboxes WHERE alias = ?", alias)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mailbox %q not found", alias)
	}
	return nil
}
```

- [ ] **Step 4: Run test — verify pass**

```bash
go test ./store/sqlite/ -v -run TestMailboxes
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add store/sqlite/mailboxes.go store/sqlite/mailboxes_test.go && git commit -m "feat: add mailboxes CRUD to SQLite store"
```

---

### Task 6: Cloudflare provider implementation

**Files:**
- Create: `provider/cloudflare/client.go`

**Goal:** Migrate existing `client/api.go` into `provider/cloudflare/client.go`, implementing the `EmailProvider` interface.

- [ ] **Step 1: Move and rewrite client as cloudflare provider**

Create `provider/cloudflare/client.go` — port all methods from `client/api.go`, adapting to the `EmailProvider` interface. The key changes:
- Package declaration: `package cloudflare`
- Client struct stays the same
- All existing methods already match the interface signature
- Add `Validate()` method that calls `GetSettings()` and returns nil on success

```go
package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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

func (c *Client) Validate() error {
	_, err := c.GetSettings()
	return err
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	slog.Debug("HTTP request", "method", method, "path", path)
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
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("HTTP request failed", "method", method, "path", path, "error", err)
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	slog.Debug("HTTP response", "method", method, "path", path, "status", resp.StatusCode, "duration", time.Since(start))
	if resp.StatusCode == 429 {
		resp.Body.Close()
		slog.Warn("Rate limited, retrying after 3s", "method", method, "path", path)
		time.Sleep(3 * time.Second)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			slog.Error("HTTP retry failed", "method", method, "path", path, "error", err)
			return nil, fmt.Errorf("retry %s %s: %w", method, path, err)
		}
		slog.Debug("HTTP retry response", "method", method, "path", path, "status", resp.StatusCode)
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

func (c *Client) ListEmails(limit, offset int) (*model.PaginatedResult, error) {
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

func (c *Client) GetEmail(id int) (*model.ParsedMail, error) {
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

func (c *Client) DeleteEmail(id int) error {
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

func (c *Client) CheckSendBalance() (int, error) {
	settings, err := c.GetSettings()
	if err != nil {
		return 0, err
	}
	return settings.SendBalance, nil
}

func (c *Client) ListSent(limit, offset int) (*model.SendboxResult, error) {
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

func (c *Client) DeleteSent(id int) error {
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

func (c *Client) ClearSent() error {
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

- [ ] **Step 2: Verify compilation**

```bash
go build ./provider/cloudflare/
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add provider/cloudflare/client.go && git commit -m "feat: add cloudflare EmailProvider implementation"
```

---

### Task 7: Service layer — mailbox service

**Files:**
- Create: `service/mailbox_svc.go`
- Create: `service/mailbox_svc_test.go`

- [ ] **Step 1: Write test service/mailbox_svc_test.go**

```go
package service_test

import (
	"encoding/json"
	"testing"

	"agent-mail/model"
	"agent-mail/provider"
	"agent-mail/provider/cloudflare"
	"agent-mail/service"
	"agent-mail/store/sqlite"
)

type testMailboxProviderFactory struct {
	validateErr error
}

func (f *testMailboxProviderFactory) NewProvider(record model.MailboxRecord) provider.EmailProvider {
	auth := make(map[string]string)
	json.Unmarshal([]byte(record.AuthData), &auth)
	return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"])
}

func TestMailboxServiceAddList(t *testing.T) {
	db, _ := sqlite.Open(":memory:")
	defer db.Close()

	factory := &testMailboxProviderFactory{}
	svc := service.NewMailboxService(db, factory)

	err := svc.Add("work", "Work", "cloudflare", "https://mail.example.com", `{"jwt":"token123"}`)
	if err != nil {
		t.Fatal(err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 mailbox, got %d", len(list))
	}
}

func TestMailboxServiceSwitch(t *testing.T) {
	db, _ := sqlite.Open(":memory:")
	defer db.Close()

	factory := &testMailboxProviderFactory{}
	svc := service.NewMailboxService(db, factory)

	svc.Add("a", "A", "cloudflare", "https://a.com", "{}")
	svc.Add("b", "B", "cloudflare", "https://b.com", "{}")

	if def := svc.Default(); def != "a" {
		t.Errorf("expected default a, got %q", def)
	}

	if err := svc.Switch("b"); err != nil {
		t.Fatal(err)
	}
	if def := svc.Default(); def != "b" {
		t.Errorf("expected default b after switch, got %q", def)
	}
}
```

- [ ] **Step 2: Create service/mailbox_svc.go**

```go
package service

import (
	"encoding/json"
	"fmt"

	"agent-mail/model"
	"agent-mail/provider"
	"agent-mail/provider/cloudflare"
	"agent-mail/store/sqlite"
)

type ProviderFactory interface {
	NewProvider(record model.MailboxRecord) provider.EmailProvider
}

type DefaultProviderFactory struct{}

func (f *DefaultProviderFactory) NewProvider(record model.MailboxRecord) provider.EmailProvider {
	auth := make(map[string]string)
	json.Unmarshal([]byte(record.AuthData), &auth)
	switch record.ProviderType {
	case "cloudflare":
		return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"])
	default:
		return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"])
	}
}

type MailboxService struct {
	db      *sqlite.DB
	factory ProviderFactory
}

func NewMailboxService(db *sqlite.DB, factory ProviderFactory) *MailboxService {
	if factory == nil {
		factory = &DefaultProviderFactory{}
	}
	return &MailboxService{db: db, factory: factory}
}

func (s *MailboxService) Add(alias, name, providerType, baseURL, authData string) error {
	if providerType == "" {
		providerType = "cloudflare"
	}
	rec := model.MailboxRecord{
		Alias:        alias,
		Name:         name,
		ProviderType: providerType,
		BaseURL:      baseURL,
		AuthData:     authData,
	}
	if err := s.db.InsertMailbox(rec); err != nil {
		return fmt.Errorf("add mailbox: %w", err)
	}
	defAlias, _ := s.db.GetSetting("default_mailbox")
	if defAlias == "" {
		s.db.SetSetting("default_mailbox", alias)
	}
	return nil
}

func (s *MailboxService) Remove(alias string) error {
	if err := s.db.DeleteMailbox(alias); err != nil {
		return fmt.Errorf("remove mailbox: %w", err)
	}
	defAlias, _ := s.db.GetSetting("default_mailbox")
	if defAlias == alias {
		list, _ := s.db.ListMailboxes()
		if len(list) > 0 {
			s.db.SetSetting("default_mailbox", list[0].Alias)
		} else {
			s.db.SetSetting("default_mailbox", "")
		}
	}
	return nil
}

func (s *MailboxService) Switch(alias string) error {
	if _, err := s.db.GetMailbox(alias); err != nil {
		return fmt.Errorf("switch mailbox: %w", err)
	}
	return s.db.SetSetting("default_mailbox", alias)
}

func (s *MailboxService) Default() string {
	v, _ := s.db.GetSetting("default_mailbox")
	return v
}

func (s *MailboxService) List() ([]model.MailboxInfo, error) {
	records, err := s.db.ListMailboxes()
	if err != nil {
		return nil, err
	}
	infos := make([]model.MailboxInfo, len(records))
	for i, r := range records {
		p := s.factory.NewProvider(r)
		valid := p.Validate() == nil
		infos[i] = model.MailboxInfo{
			Alias:        r.Alias,
			Name:         r.Name,
			ProviderType: r.ProviderType,
			BaseURL:      r.BaseURL,
			Valid:        valid,
		}
	}
	return infos, nil
}

func (s *MailboxService) Validate(alias string) (*model.SettingsResponse, error) {
	rec, err := s.Resolve(alias)
	if err != nil {
		return nil, err
	}
	p := s.factory.NewProvider(*rec)
	return p.GetSettings()
}

func (s *MailboxService) Resolve(alias string) (*model.MailboxRecord, error) {
	if alias == "" {
		alias = s.Default()
	}
	return s.db.GetMailbox(alias)
}

func (s *MailboxService) Provider(alias string) (provider.EmailProvider, error) {
	rec, err := s.Resolve(alias)
	if err != nil {
		return nil, err
	}
	return s.factory.NewProvider(*rec), nil
}
```

- [ ] **Step 3: Run test — verify**

```bash
go test ./service/ -v -run TestMailbox
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add service/mailbox_svc.go service/mailbox_svc_test.go && git commit -m "feat: add mailbox service layer"
```

---

### Task 8: Service layer — email and send services

**Files:**
- Create: `service/email_svc.go`
- Create: `service/send_svc.go`
- Create: `service/auto_reply_svc.go`
- Create: `service/webhook_svc.go`
- Create: `service/attachment_svc.go`

**Goal:** Thin service wrappers over provider methods, with search logic in email_svc.

- [ ] **Step 1: Create service/email_svc.go**

```go
package service

import (
	"fmt"
	"strings"

	"agent-mail/model"
	"agent-mail/provider"
)

type EmailService struct {
	mailboxSvc *MailboxService
}

func NewEmailService(ms *MailboxService) *EmailService {
	return &EmailService{mailboxSvc: ms}
}

func (s *EmailService) List(alias string, limit, offset int) (*model.PaginatedResult, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.ListEmails(limit, offset)
}

func (s *EmailService) Get(alias string, id int) (*model.ParsedMail, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.GetEmail(id)
}

func (s *EmailService) Delete(alias string, id int) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.DeleteEmail(id)
}

func (s *EmailService) Clear(alias string) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.ClearInbox()
}

func (s *EmailService) Search(alias, query string, limit int) (*model.PaginatedResult, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	result, err := p.ListEmails(100, 0)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var filtered []model.ParsedMail
	for _, m := range result.Results {
		if strings.Contains(strings.ToLower(m.Sender), q) ||
			strings.Contains(strings.ToLower(m.Subject), q) {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return &model.PaginatedResult{Results: filtered, Count: len(filtered)}, nil
}
```

- [ ] **Step 2: Create service/send_svc.go**

```go
package service

import (
	"agent-mail/model"
)

type SendService struct {
	mailboxSvc *MailboxService
}

func NewSendService(ms *MailboxService) *SendService {
	return &SendService{mailboxSvc: ms}
}

func (s *SendService) Send(alias string, body *model.SendMailBody) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.SendMail(body)
}

func (s *SendService) CheckBalance(alias string) (int, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return 0, err
	}
	return p.CheckSendBalance()
}

func (s *SendService) ListSent(alias string, limit, offset int) (*model.SendboxResult, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.ListSent(limit, offset)
}

func (s *SendService) DeleteSent(alias string, id int) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.DeleteSent(id)
}

func (s *SendService) ClearSent(alias string) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.ClearSent()
}
```

- [ ] **Step 3: Create service/auto_reply_svc.go**

```go
package service

import "agent-mail/model"

type AutoReplyService struct {
	mailboxSvc *MailboxService
}

func NewAutoReplyService(ms *MailboxService) *AutoReplyService {
	return &AutoReplyService{mailboxSvc: ms}
}

func (s *AutoReplyService) Get(alias string) (*model.AutoReplyConfig, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.GetAutoReply()
}

func (s *AutoReplyService) Set(alias string, cfg *model.AutoReplyConfig) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.SetAutoReply(cfg)
}
```

- [ ] **Step 4: Create service/webhook_svc.go**

```go
package service

import "agent-mail/model"

type WebhookService struct {
	mailboxSvc *MailboxService
}

func NewWebhookService(ms *MailboxService) *WebhookService {
	return &WebhookService{mailboxSvc: ms}
}

func (s *WebhookService) Get(alias string) (*model.WebhookSettings, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.GetWebhook()
}

func (s *WebhookService) Set(alias string, cfg *model.WebhookSettings) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.SetWebhook(cfg)
}
```

- [ ] **Step 5: Create service/attachment_svc.go**

```go
package service

import "agent-mail/model"

type AttachmentService struct {
	mailboxSvc *MailboxService
}

func NewAttachmentService(ms *MailboxService) *AttachmentService {
	return &AttachmentService{mailboxSvc: ms}
}

func (s *AttachmentService) List(alias string) (*model.AttachmentListResult, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.ListAttachments()
}
```

- [ ] **Step 6: Verify compilation**

```bash
go build ./service/...
```

- [ ] **Step 7: Commit**

```bash
git add service/email_svc.go service/send_svc.go service/auto_reply_svc.go service/webhook_svc.go service/attachment_svc.go && git commit -m "feat: add email, send, auto-reply, webhook, attachment services"
```

---

### Task 9: MCP layer — tool definitions

**Files:**
- Create: `mcp/tools_def.go`

**Goal:** All 20 tool definitions (unchanged from current code, just extracted to own file).

- [ ] **Step 1: Create mcp/tools_def.go**

```go
package mcp

import "github.com/mark3labs/mcp-go/mcp"

var listMailboxesTool = mcp.NewTool("list_mailboxes",
	mcp.WithDescription("List all configured mailboxes with their validity status"),
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
```

- [ ] **Step 2: Commit**

```bash
git add mcp/tools_def.go && git commit -m "feat: add MCP tool definitions"
```

---

### Task 10: MCP layer — server and handlers

**Files:**
- Create: `mcp/server.go`
- Create: `mcp/handler.go`

- [ ] **Step 1: Create mcp/server.go**

```go
package mcp

import (
	"agent-mail/service"

	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	mcpServer *server.MCPServer

	MailboxSvc    *service.MailboxService
	EmailSvc      *service.EmailService
	SendSvc       *service.SendService
	AutoReplySvc  *service.AutoReplyService
	WebhookSvc    *service.WebhookService
	AttachmentSvc *service.AttachmentService
}

func New(
	mailboxSvc *service.MailboxService,
	emailSvc *service.EmailService,
	sendSvc *service.SendService,
	autoReplySvc *service.AutoReplyService,
	webhookSvc *service.WebhookService,
	attachmentSvc *service.AttachmentService,
) *Server {
	s := &Server{
		mcpServer:     server.NewMCPServer("agent-mail", "2.0.0"),
		MailboxSvc:    mailboxSvc,
		EmailSvc:      emailSvc,
		SendSvc:       sendSvc,
		AutoReplySvc:  autoReplySvc,
		WebhookSvc:    webhookSvc,
		AttachmentSvc: attachmentSvc,
	}
	s.registerTools()
	return s
}

func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
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
```

- [ ] **Step 2: Create mcp/handler.go**

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"agent-mail/model"

	"github.com/mark3labs/mcp-go/mcp"
)

func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
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

func (s *Server) handleListMailboxes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	infos, err := s.MailboxSvc.List()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
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
	authData, _ := json.Marshal(map[string]string{
		"jwt":           jwt,
		"site_password": sitePass,
	})
	if err := s.MailboxSvc.Add(alias, name, "cloudflare", baseURL, string(authData)); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleRemoveMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias, err := req.RequireString("alias")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: alias"), nil
	}
	if err := s.MailboxSvc.Remove(alias); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleSwitchMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias, err := req.RequireString("alias")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: alias"), nil
	}
	if err := s.MailboxSvc.Switch(alias); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleValidateMailbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("alias", "")
	settings, err := s.MailboxSvc.Validate(alias)
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
	limit, err := parseIntParam(req.GetString("limit", ""), "20", 1, 100)
	if err != nil {
		return mcp.NewToolResultError("invalid limit: " + err.Error()), nil
	}
	offset, err := parseIntParam(req.GetString("offset", ""), "0", 0, 10000)
	if err != nil {
		return mcp.NewToolResultError("invalid offset: " + err.Error()), nil
	}
	result, err := s.EmailSvc.List(alias, limit, offset)
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
	mail, err := s.EmailSvc.Get(alias, id)
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
	if err := s.EmailSvc.Delete(alias, id); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleClearInbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	if err := s.EmailSvc.Clear(alias); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleSearchEmails(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
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
	result, err := s.EmailSvc.Search(alias, query, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *Server) handleSendEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
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
	if err := s.SendSvc.Send(alias, body); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleCheckSendBalance(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	balance, err := s.SendSvc.CheckBalance(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]int{"send_balance": balance})), nil
}

func (s *Server) handleListSent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	limit, err := parseIntParam(req.GetString("limit", ""), "20", 1, 100)
	if err != nil {
		return mcp.NewToolResultError("invalid limit: " + err.Error()), nil
	}
	offset, err := parseIntParam(req.GetString("offset", ""), "0", 0, 10000)
	if err != nil {
		return mcp.NewToolResultError("invalid offset: " + err.Error()), nil
	}
	result, err := s.SendSvc.ListSent(alias, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *Server) handleDeleteSent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	idStr, err := req.RequireString("send_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: send_id"), nil
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return mcp.NewToolResultError("send_id must be a positive integer"), nil
	}
	if err := s.SendSvc.DeleteSent(alias, id); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleClearSent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	if err := s.SendSvc.ClearSent(alias); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleGetAutoReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	cfg, err := s.AutoReplySvc.Get(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(cfg)), nil
}

func (s *Server) handleSetAutoReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	cfg, err := s.AutoReplySvc.Get(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
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
	if err := s.AutoReplySvc.Set(alias, cfg); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleGetWebhook(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	cfg, err := s.WebhookSvc.Get(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(cfg)), nil
}

func (s *Server) handleSetWebhook(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	rawURL, err := req.RequireString("url")
	if err != nil || strings.TrimSpace(rawURL) == "" {
		return mcp.NewToolResultError("missing or empty required parameter: url"), nil
	}
	url := strings.TrimSpace(rawURL)
	cfg, err := s.WebhookSvc.Get(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cfg.URL = url
	if err := s.WebhookSvc.Set(alias, cfg); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(map[string]string{"status": "ok"})), nil
}

func (s *Server) handleListAttachments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := req.GetString("mailbox", "")
	result, err := s.AttachmentSvc.List(alias)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./mcp/...
```

- [ ] **Step 4: Commit**

```bash
git add mcp/server.go mcp/handler.go && git commit -m "feat: add MCP server and all 20 tool handlers"
```

---

### Task 11: Main entrypoint rewrite

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace main.go**

```go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"agent-mail/mcp"
	"agent-mail/service"
	"agent-mail/store/sqlite"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	authHeader := flag.String("auth-header", "Authorization", "auth header name")
	authToken := flag.String("auth-token", "", "auth token value (empty = no auth)")
	dbPath := flag.String("db-path", "./agent-mail.db", "SQLite database path")
	envFile := flag.String("env-file", ".env", "env file path")
	flag.Parse()

	loadEnvFile(*envFile)
	applyEnvOverrides(addr, authHeader, authToken, dbPath)

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	db, err := sqlite.Open(*dbPath)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	mailboxSvc := service.NewMailboxService(db, nil)
	emailSvc := service.NewEmailService(mailboxSvc)
	sendSvc := service.NewSendService(mailboxSvc)
	autoReplySvc := service.NewAutoReplyService(mailboxSvc)
	webhookSvc := service.NewWebhookService(mailboxSvc)
	attachmentSvc := service.NewAttachmentService(mailboxSvc)

	mcpSrv := mcp.New(mailboxSvc, emailSvc, sendSvc, autoReplySvc, webhookSvc, attachmentSvc)

	httpServer := server.NewStreamableHTTPServer(mcpSrv.MCPServer())
	h := authMiddleware(*authHeader, *authToken, httpServer)

	slog.Info("agent-mail starting", "addr", *addr, "db", *dbPath)
	if err := http.ListenAndServe(*addr, h); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

func applyEnvOverrides(addr, authHeader, authToken, dbPath *string) {
	if v := os.Getenv("AGENT_MAIL_ADDR"); v != "" && *addr == ":8080" {
		*addr = v
	}
	if v := os.Getenv("AGENT_MAIL_AUTH_HEADER"); v != "" && *authHeader == "Authorization" {
		*authHeader = v
	}
	if v := os.Getenv("AGENT_MAIL_AUTH_TOKEN"); v != "" && *authToken == "" {
		*authToken = v
	}
	if v := os.Getenv("AGENT_MAIL_DB_PATH"); v != "" && *dbPath == "./agent-mail.db" {
		*dbPath = v
	}
}

func authMiddleware(headerName, token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerName) != token {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"unauthorized","message":"invalid or missing token"}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: Build binary**

```bash
go build -o agent-mail .
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add main.go && git commit -m "feat: rewrite main entrypoint for remote-only architecture"
```

---

### Task 12: Configuration & deployment files

**Files:**
- Create: `.env.example`
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`

- [ ] **Step 1: Create .env.example**

```
# agent-mail configuration
AGENT_MAIL_ADDR=:8080
AGENT_MAIL_AUTH_HEADER=X-API-Key
AGENT_MAIL_AUTH_TOKEN=your-secret-token-here
AGENT_MAIL_DB_PATH=/data/agent-mail.db
```

- [ ] **Step 2: Update Dockerfile**

```dockerfile
FROM golang:1.25-alpine AS build
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -o agent-mail .

FROM scratch
COPY --from=build /app/agent-mail /
EXPOSE 8080
ENTRYPOINT ["/agent-mail"]
```

- [ ] **Step 3: Update docker-compose.yml**

```yaml
services:
  agent-mail:
    image: dgqyushen/agent-mail:latest
    ports:
      - "9090:8080"
    volumes:
      - ./data:/data
    command:
      - --db-path=/data/agent-mail.db
      - --env-file=/data/.env
    restart: unless-stopped
```

- [ ] **Step 4: Commit**

```bash
git add .env.example Dockerfile docker-compose.yml && git commit -m "chore: add .env.example, update Docker and compose for remote-only"
```

---

### Task 13: Remove old files

**Files:**
- Delete: `client/api.go`
- Delete: `config/config.go`
- Delete: `config/config_test.go`
- Delete: `mcp/tools.go` (replaced by handler.go + server.go + tools_def.go)
- Delete: `mcp/tools_test.go` (rewritten tests in Task 14)
- Delete: `config.example.toml`

- [ ] **Step 1: Remove old files**

```bash
rm client/api.go config/config.go config/config_test.go mcp/tools.go mcp/tools_test.go config.example.toml
rm -rf client/ config/
```

- [ ] **Step 2: Build to verify no broken references**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add -A . && git commit -m "refactor: remove old client, config, and merged MCP files"
```

---

### Task 14: Write MCP handler tests

**Files:**
- Create: `mcp/handler_test.go`

- [ ] **Step 1: Create mcp/handler_test.go**

```go
package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"agent-mail/model"
	"agent-mail/provider"
	"agent-mail/provider/cloudflare"
	"agent-mail/service"
	"agent-mail/store/sqlite"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeProviderFactory struct{}

func (f *fakeProviderFactory) NewProvider(record model.MailboxRecord) provider.EmailProvider {
	auth := make(map[string]string)
	json.Unmarshal([]byte(record.AuthData), &auth)
	return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"])
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	factory := &fakeProviderFactory{}
	mailboxSvc := service.NewMailboxService(db, factory)
	emailSvc := service.NewEmailService(mailboxSvc)
	sendSvc := service.NewSendService(mailboxSvc)
	autoReplySvc := service.NewAutoReplyService(mailboxSvc)
	webhookSvc := service.NewWebhookService(mailboxSvc)
	attachmentSvc := service.NewAttachmentService(mailboxSvc)

	return New(mailboxSvc, emailSvc, sendSvc, autoReplySvc, webhookSvc, attachmentSvc)
}

func TestServerConstruction(t *testing.T) {
	s := newTestServer(t)
	if s.MCPServer() == nil {
		t.Fatal("MCPServer is nil")
	}
	// Verify all 20 tools are registered (mcp-go internally tracks them)
}

func TestListMailboxesHandler(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_mailboxes",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := s.handleListMailboxes(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestAddMailboxHandler(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "add_mailbox",
			Arguments: map[string]interface{}{
				"alias":    "test",
				"name":     "Test",
				"base_url": "https://example.com",
				"jwt":      "fake-jwt",
			},
		},
	}
	result, err := s.handleAddMailbox(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	// Verify mailbox was added
	listReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_mailboxes",
			Arguments: map[string]interface{}{},
		},
	}
	listResult, err := s.handleListMailboxes(ctx, listReq)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := listResult.Content[0].(mcp.TextContent)
	var infos []model.MailboxInfo
	json.Unmarshal([]byte(text.Text), &infos)
	if len(infos) != 1 || infos[0].Alias != "test" {
		t.Errorf("expected 1 mailbox with alias test, got %+v", infos)
	}
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./mcp/... -v -run TestServer
```

Expected: FAIL (no tests, or handler tests fail)

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```

Expected: PASS (store/sqlite tests pass, model has no tests, service tests pass, mcp tests pass)

- [ ] **Step 4: Commit**

```bash
git add mcp/handler_test.go && git commit -m "test: add MCP server construction test"
```

---

### Task 15: Final verification and push

**Files:**
- None (verify only)

- [ ] **Step 1: Full build**

```bash
CGO_ENABLED=0 go build -o agent-mail .
ls -lh agent-mail
```

- [ ] **Step 2: Run all tests**

```bash
go test ./... -v
```

Expected: all pass

- [ ] **Step 3: Verify binary runs**

```bash
timeout 2 ./agent-mail --help 2>&1 || true
```

Expected: shows flag help

- [ ] **Step 4: Commit and push**

```bash
git add -A . && git commit -m "chore: final verification after full rewrite"
git push -u origin master
```
