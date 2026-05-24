# agent-mail Multi-User Token Management 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 agent-mail 从单用户改为多用户系统，支持 token 认证、用户隔离、admin 面板

**Architecture:** 新增 users/tokens 表实现用户隔离，token 存 SHA256 hash；`X-Agent-Mail-Token` header 认证；admin 面板用 Go embed template 嵌入二进制

**Tech Stack:** Go 1.25, modernc.org/sqlite, golang.org/x/crypto (bcrypt), embed, html/template, mark3labs/mcp-go

---

### Task 1: DB migration — 新增 users/tokens 表 + mailboxes 加 user_id

**Files:**
- Modify: `store/sqlite/db.go:42-58`

- [ ] **Step 1: 更新 migrate 函数**

```go
func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS tokens (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			token_hash TEXT NOT NULL UNIQUE,
			prefix     TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			is_active  INTEGER NOT NULL DEFAULT 1
		);

		CREATE TABLE IF NOT EXISTS mailboxes (
			alias          TEXT NOT NULL,
			user_id        INTEGER NOT NULL DEFAULT 0,
			name           TEXT NOT NULL,
			provider_type  TEXT NOT NULL DEFAULT 'cloudflare',
			base_url       TEXT NOT NULL,
			auth_data      TEXT NOT NULL DEFAULT '{}',
			created_at     TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at     TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (user_id, alias)
		);
	`)
	return err
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./store/...
```

- [ ] **Step 3: Commit**

```bash
git add store/sqlite/db.go && git commit -m "feat: add users/tokens tables, user_id to mailboxes"
```

---

### Task 2: store/sqlite/users.go — 用户 CRUD

**Files:**
- Create: `store/sqlite/users.go`
- Create: `store/sqlite/users_test.go`

- [ ] **Step 1: 创建 users.go**

```go
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type User struct {
	ID        int
	Name      string
	CreatedAt string
}

func (db *DB) CreateUser(name string) (*User, error) {
	result, err := db.conn.Exec("INSERT INTO users (name) VALUES (?)", name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, fmt.Errorf("user %q already exists", name)
		}
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &User{ID: int(id), Name: name}, nil
}

func (db *DB) GetUser(id int) (*User, error) {
	var u User
	err := db.conn.QueryRow(
		"SELECT id, name, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user %d not found", id)
		}
		return nil, err
	}
	return &u, nil
}

func (db *DB) ListUsers() ([]User, error) {
	rows, err := db.conn.Query("SELECT id, name, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *DB) DeleteUser(id int) error {
	result, err := db.conn.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user %d not found", id)
	}
	return nil
}
```

- [ ] **Step 2: 创建 users_test.go**

```go
package sqlite_test

import (
	"testing"
	"agent-mail/store/sqlite"
)

func TestUsersCRUD(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	u, err := db.CreateUser("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 || u.Name != "Alice" {
		t.Errorf("unexpected user: %+v", u)
	}

	got, err := db.GetUser(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Alice" {
		t.Errorf("expected Alice, got %q", got.Name)
	}

	list, err := db.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 user, got %d", len(list))
	}
}

func TestUserNotFound(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.GetUser(999)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteUser(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.CreateUser("Bob")
	if err := db.DeleteUser(1); err != nil {
		t.Fatal(err)
	}
	_, err = db.GetUser(1)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./store/sqlite/... -v -run TestUser
```

- [ ] **Step 4: Commit**

```bash
git add store/sqlite/users.go store/sqlite/users_test.go && git commit -m "feat: add user CRUD store"
```

---

### Task 3: store/sqlite/tokens.go — Token CRUD

**Files:**
- Create: `store/sqlite/tokens.go`
- Create: `store/sqlite/tokens_test.go`

- [ ] **Step 1: 创建 tokens.go**

```go
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
)

type Token struct {
	ID        int
	UserID    int
	TokenHash string
	Prefix    string
	CreatedAt string
	IsActive  bool
}

func (db *DB) InsertToken(userID int, tokenHash, prefix string) (*Token, error) {
	result, err := db.conn.Exec(
		"INSERT INTO tokens (user_id, token_hash, prefix) VALUES (?, ?, ?)",
		userID, tokenHash, prefix,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Token{ID: int(id), UserID: userID, TokenHash: tokenHash, Prefix: prefix, IsActive: true}, nil
}

func (db *DB) FindActiveToken(tokenHash string) (*Token, error) {
	var t Token
	err := db.conn.QueryRow(
		`SELECT id, user_id, token_hash, prefix, created_at, is_active
		 FROM tokens WHERE token_hash = ? AND is_active = 1`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Prefix, &t.CreatedAt, &t.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (db *DB) GetActiveToken(userID int) (*Token, error) {
	var t Token
	err := db.conn.QueryRow(
		`SELECT id, user_id, token_hash, prefix, created_at, is_active
		 FROM tokens WHERE user_id = ? AND is_active = 1`, userID,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Prefix, &t.CreatedAt, &t.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (db *DB) DeactivateTokens(userID int) error {
	_, err := db.conn.Exec("UPDATE tokens SET is_active = 0 WHERE user_id = ? AND is_active = 1", userID)
	return err
}
```

- [ ] **Step 2: 创建 tokens_test.go**

```go
package sqlite_test

import (
	"testing"
	"agent-mail/store/sqlite"
)

func TestTokenInsertFind(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.CreateUser("TestUser")

	tok, err := db.InsertToken(1, "abc_hash", "atm-abc***")
	if err != nil {
		t.Fatal(err)
	}
	if tok.UserID != 1 || tok.Prefix != "atm-abc***" {
		t.Errorf("unexpected token: %+v", tok)
	}

	found, err := db.FindActiveToken("abc_hash")
	if err != nil {
		t.Fatal(err)
	}
	if found == nil {
		t.Fatal("expected token found")
	}
	if found.UserID != 1 {
		t.Errorf("expected user 1, got %d", found.UserID)
	}
}

func TestTokenDeactivate(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.CreateUser("TestUser")
	db.InsertToken(1, "old_hash", "atm-old***")

	if err := db.DeactivateTokens(1); err != nil {
		t.Fatal(err)
	}

	found, err := db.FindActiveToken("old_hash")
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatal("expected token to be deactivated")
	}
}

func TestGetActiveToken(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.CreateUser("TestUser")
	db.InsertToken(1, "hash1", "atm-000***")

	tok, err := db.GetActiveToken(1)
	if err != nil {
		t.Fatal(err)
	}
	if tok == nil {
		t.Fatal("expected active token")
	}

	db.DeactivateTokens(1)
	tok, err = db.GetActiveToken(1)
	if err != nil {
		t.Fatal(err)
	}
	if tok != nil {
		t.Fatal("expected nil after deactivate")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./store/sqlite/... -v -run TestToken
```

- [ ] **Step 4: Commit**

```bash
git add store/sqlite/tokens.go store/sqlite/tokens_test.go && git commit -m "feat: add token CRUD store"
```

---

### Task 4: store/sqlite/mailboxes.go — MailboxRecord 增加 user_id

**Files:**
- Modify: `store/sqlite/mailboxes.go` (全部方法)
- Modify: `model/types.go:90-96` (MailboxRecord 加 UserID)
- Modify: `store/sqlite/mailboxes_test.go` (适配)

- [ ] **Step 1: 更新 model/types.go — MailboxRecord 加 UserID**

```go
type MailboxRecord struct {
	UserID       int
	Alias        string
	Name         string
	ProviderType string
	BaseURL      string
	AuthData     string
}
```

- [ ] **Step 2: 更新 mailboxes.go — 所有 CRUD 加 user_id**

```go
func (db *DB) InsertMailbox(m model.MailboxRecord) error {
	_, err := db.conn.Exec(
		`INSERT INTO mailboxes (user_id, alias, name, provider_type, base_url, auth_data)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		m.UserID, m.Alias, m.Name, m.ProviderType, m.BaseURL, m.AuthData,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return fmt.Errorf("mailbox %q already exists", m.Alias)
		}
		return err
	}
	return nil
}

func (db *DB) GetMailbox(userID int, alias string) (*model.MailboxRecord, error) {
	var m model.MailboxRecord
	err := db.conn.QueryRow(
		`SELECT user_id, alias, name, provider_type, base_url, auth_data
		 FROM mailboxes WHERE user_id = ? AND alias = ?`,
		userID, alias,
	).Scan(&m.UserID, &m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("mailbox %q not found", alias)
		}
		return nil, err
	}
	if m.ProviderType == "" {
		m.ProviderType = "cloudflare"
	}
	return &m, nil
}

func (db *DB) ListMailboxes(userID int) ([]model.MailboxRecord, error) {
	rows, err := db.conn.Query(
		`SELECT user_id, alias, name, provider_type, base_url, auth_data
		 FROM mailboxes WHERE user_id = ? ORDER BY alias`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.MailboxRecord
	for rows.Next() {
		var m model.MailboxRecord
		if err := rows.Scan(&m.UserID, &m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData); err != nil {
			return nil, err
		}
		if m.ProviderType == "" {
			m.ProviderType = "cloudflare"
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (db *DB) DeleteMailbox(userID int, alias string) error {
	result, err := db.conn.Exec(
		"DELETE FROM mailboxes WHERE user_id = ? AND alias = ?", userID, alias,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("mailbox %q not found", alias)
	}
	return nil
}
```

- [ ] **Step 3: 更新测试中的调用（所有 GetMailbox/ListMailboxes/InsertMailbox/DeleteMailbox 加 userID 参数）**

```go
// 所有 test 中的 MailboxRecord 加 UserID: 1
// 所有 db.InsertMailbox/GetMailbox/ListMailboxes/DeleteMailbox 调用加第一个参数 1
```

- [ ] **Step 4: 运行测试**

```bash
go test ./store/sqlite/... -v
```

- [ ] **Step 5: Commit**

```bash
git add model/types.go store/sqlite/mailboxes.go store/sqlite/mailboxes_test.go && git commit -m "feat: add user_id to mailbox CRUD for multi-tenant isolation"
```

---

### Task 5: 安装 bcrypt 依赖 + service/user_svc.go — 用户 + Token 服务

**Files:**
- Create: `service/user_svc.go`
- Create: `service/user_svc_test.go`

- [ ] **Step 1: 安装依赖**

```bash
go get golang.org/x/crypto/bcrypt
```

```bash
git add go.mod go.sum && git commit -m "chore: add bcrypt dependency"
```

- [ ] **Step 2: 创建 user_svc.go**

```go
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"agent-mail/store/sqlite"
)

type UserService struct {
	db *sqlite.DB
}

func NewUserService(db *sqlite.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) CreateUser(name string) (*sqlite.User, string, error) {
	u, err := s.db.CreateUser(name)
	if err != nil {
		return nil, "", err
	}
	token, err := s.generateToken(u.ID)
	if err != nil {
		return nil, "", fmt.Errorf("create user but failed to generate token: %w", err)
	}
	return u, token, nil
}

func (s *UserService) RefreshToken(userID int) (string, error) {
	if err := s.db.DeactivateTokens(userID); err != nil {
		return "", fmt.Errorf("deactivate old tokens: %w", err)
	}
	return s.generateToken(userID)
}

func (s *UserService) ValidateToken(token string) (int, error) {
	hash := sha256Hex(token)
	t, err := s.db.FindActiveToken(hash)
	if err != nil {
		return 0, err
	}
	if t == nil {
		return 0, fmt.Errorf("invalid or expired token")
	}
	return t.UserID, nil
}

func (s *UserService) GetActiveTokenPrefix(userID int) string {
	t, err := s.db.GetActiveToken(userID)
	if err != nil || t == nil {
		return ""
	}
	return t.Prefix
}

func (s *UserService) ListUsers() ([]sqlite.User, error) {
	return s.db.ListUsers()
}

func (s *UserService) GetUser(id int) (*sqlite.User, error) {
	return s.db.GetUser(id)
}

func (s *UserService) DeleteUser(id int) error {
	s.db.DeactivateTokens(id)
	return s.db.DeleteUser(id)
}

func (s *UserService) generateToken(userID int) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := "atm-" + hex.EncodeToString(b)
	prefix := token[:11] + "***"
	hash := sha256Hex(token)
	_, err := s.db.InsertToken(userID, hash, prefix)
	return token, err
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 3: 创建 user_svc_test.go**

```go
package service_test

import (
	"testing"

	"agent-mail/service"
	"agent-mail/store/sqlite"
)

func TestUserServiceCreateAndValidateToken(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)

	u, token, err := svc.CreateUser("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if u.Name != "Alice" {
		t.Errorf("expected Alice, got %q", u.Name)
	}
	if len(token) < 10 || token[:4] != "atm-" {
		t.Errorf("invalid token format: %q", token)
	}

	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if userID != u.ID {
		t.Errorf("expected userID %d, got %d", u.ID, userID)
	}
}

func TestUserServiceRefreshToken(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)

	u, oldToken, err := svc.CreateUser("Bob")
	if err != nil {
		t.Fatal(err)
	}

	newToken, err := svc.RefreshToken(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newToken == oldToken {
		t.Fatal("refresh should produce new token")
	}

	// Old token should be invalid
	_, err = svc.ValidateToken(oldToken)
	if err == nil {
		t.Fatal("old token should be invalid after refresh")
	}

	// New token should work
	uid, err := svc.ValidateToken(newToken)
	if err != nil {
		t.Fatal(err)
	}
	if uid != u.ID {
		t.Errorf("expected userID %d, got %d", u.ID, uid)
	}
}

func TestUserServiceInvalidToken(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)
	_, err = svc.ValidateToken("atm-badtoken")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestUserServiceListAndDelete(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)
	svc.CreateUser("A")
	svc.CreateUser("B")

	users, err := svc.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	if err := svc.DeleteUser(1); err != nil {
		t.Fatal(err)
	}
	users, _ = svc.ListUsers()
	if len(users) != 1 {
		t.Errorf("expected 1 user after delete, got %d", len(users))
	}
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./service/... -v -run TestUser
```

- [ ] **Step 5: Commit**

```bash
git add service/user_svc.go service/user_svc_test.go && git commit -m "feat: add UserService with token generation and validation"
```

---

### Task 6: mcp/auth.go — 重写中间件，查 tokens 表

**Files:**
- Modify: `mcp/auth.go` (全部重写)

- [ ] **Step 1: 重写 auth.go**

```go
package mcp

import (
	"context"
	"net/http"
	"os"

	"agent-mail/service"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func GetUserID(ctx context.Context) int {
	v, _ := ctx.Value(UserIDKey).(int)
	return v
}

func authMiddleware(next http.Handler, userSvc *service.UserService) http.Handler {
	legacyHeader := os.Getenv("AUTH_HEADER")
	legacyToken := os.Getenv("AUTH_TOKEN")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Agent-Mail-Token")
		if token == "" && legacyHeader != "" && legacyToken != "" {
			token = r.Header.Get(legacyHeader)
		}

		var userID int
		if token != "" {
			uid, err := userSvc.ValidateToken(token)
			if err != nil {
				// Try legacy token
				if legacyHeader != "" && token == legacyToken {
					userID = 0
				} else {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"unauthorized: invalid token"}`))
					return
				}
			}
			userID = uid
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized: missing X-Agent-Mail-Token header"}`))
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

- [ ] **Step 2: 更新 mcp/server.go — 传入 userSvc**

将 `authMiddleware(streamableServer)` 改为 `authMiddleware(streamableServer, userSvc)`。
必须在 `NewServer` 中接收 `*service.UserService` 参数并保存。

```go
type Server struct {
	httpServer *http.Server
	handler    *Handler
	userSvc    *service.UserService
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
		userSvc:  userSvc,
	}
}

func (s *Server) Start() error {
	// ...mcpSrv setup same as before...
	mux.Handle("/mcp", authMiddleware(streamableServer, s.userSvc))
	// ...
}
```

- [ ] **Step 3: 编译验证**

```bash
go build ./mcp/...
```

- [ ] **Step 4: Commit**

```bash
git add mcp/auth.go mcp/server.go && git commit -m "feat: rewrite auth middleware for multi-user token validation"
```

---

### Task 7: mcp/handler.go + service/mailbox_svc.go — 用户隔离

**Files:**
- Modify: `mcp/handler.go` (所有 handler 从 context 取 user_id)
- Modify: `service/mailbox_svc.go` (所有方法加 userID 参数)
- Modify: `service/email_svc.go`, `service/send_svc.go`, `service/auto_reply_svc.go`, `service/webhook_svc.go`, `service/attachment_svc.go` (透传 userID)

- [ ] **Step 1: 更新 service/mailbox_svc.go**

所有方法加 `userID int` 作为第一个参数。settings key 改为 `default_mailbox_<userID>`：

```go
func (s *MailboxService) Add(userID int, alias, name, providerType, baseURL, authData string) error {
	// ...
	rec := model.MailboxRecord{
		UserID:       userID,
		Alias:        alias,
		// ...
	}
	if err := s.db.InsertMailbox(rec); err != nil {
		return fmt.Errorf("add mailbox: %w", err)
	}
	dk := fmt.Sprintf("default_mailbox_%d", userID)
	defAlias, _ := s.db.GetSetting(dk)
	if defAlias == "" {
		s.db.SetSetting(dk, alias)
	}
	return nil
}

func (s *MailboxService) Remove(userID int, alias string) error {
	if err := s.db.DeleteMailbox(userID, alias); err != nil {
		return fmt.Errorf("remove mailbox: %w", err)
	}
	dk := fmt.Sprintf("default_mailbox_%d", userID)
	// ...same fallback logic with dk key...
}

func (s *MailboxService) Switch(userID int, alias string) error {
	if _, err := s.db.GetMailbox(userID, alias); err != nil {
		return fmt.Errorf("switch mailbox: %w", err)
	}
	return s.db.SetSetting(fmt.Sprintf("default_mailbox_%d", userID), alias)
}

func (s *MailboxService) Default(userID int) string {
	v, _ := s.db.GetSetting(fmt.Sprintf("default_mailbox_%d", userID))
	return v
}

func (s *MailboxService) List(userID int) ([]model.MailboxInfo, error) {
	records, err := s.db.ListMailboxes(userID)
	// ...same logic...
}

func (s *MailboxService) Resolve(userID int, alias string) (*model.MailboxRecord, error) {
	if alias == "" {
		alias = s.Default(userID)
	}
	return s.db.GetMailbox(userID, alias)
}

func (s *MailboxService) Provider(userID int, alias string) (provider.EmailProvider, error) {
	rec, err := s.Resolve(userID, alias)
	if err != nil {
		return nil, err
	}
	return s.factory.NewProvider(*rec)
}

func (s *MailboxService) Validate(userID int, alias string) (*model.SettingsResponse, error) {
	rec, err := s.Resolve(userID, alias)
	if err != nil {
		return nil, err
	}
	p, err := s.factory.NewProvider(*rec)
	if err != nil {
		return nil, err
	}
	return p.GetSettings()
}
```

- [ ] **Step 2: 更新其他 service 文件 — 所有方法加 userID 透传**

email_svc.go, send_svc.go, auto_reply_svc.go, webhook_svc.go, attachment_svc.go 中每个方法的 `alias` 参数前加 `userID int`。所有 `s.mailboxSvc.Provider(alias)` 改为 `s.mailboxSvc.Provider(userID, alias)`。

- [ ] **Step 3: 更新 mcp/handler.go — 所有 handler 从 context 取 user_id**

```go
func (h *Handler) HandleToolCall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID := GetUserID(ctx)
	// 所有 handler 函数加 userID 参数，传递给 service
	switch req.Params.Name {
	case ToolListMailboxes:
		return h.listMailboxes(userID)
	// ...
	}
}

func (h *Handler) listMailboxes(userID int) (*mcp.CallToolResult, error) {
	list, err := h.mailbox.List(userID)
	// ...
}

func (h *Handler) addMailbox(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "alias", "name", "base_url", "auth_data"); err != nil {
		return errorResult(err), nil
	}
	err := h.mailbox.Add(
		userID,
		strArg(args, "alias"),
		// ...
	)
	// ...
}
```

（所有 20 个 handler 同样处理）

- [ ] **Step 4: 编译验证**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add service/ mcp/handler.go && git commit -m "feat: add user isolation — all services require user_id"
```

---

### Task 8: web/ — Admin 前端（嵌入模板 + handler）

**Files:**
- Create: `web/templates/base.html`
- Create: `web/templates/login.html`
- Create: `web/templates/users.html`
- Create: `web/templates/user_detail.html`
- Create: `web/templates/user_create.html`
- Create: `web/handler.go`
- Create: `web/session.go`

- [ ] **Step 1: 创建 web/session.go**

```go
package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

type sessionStore struct {
	tokens map[string]time.Time
}

var sessions = &sessionStore{tokens: make(map[string]time.Time)}

func newSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

const sessionCookie = "admin_session"
const sessionTTL = 12 * time.Hour

func setSession(w http.ResponseWriter) string {
	sid := newSession()
	sessions.tokens[sid] = time.Now().Add(sessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sid,
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return sid
}

func checkSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	exp, ok := sessions.tokens[c.Value]
	if !ok || time.Now().After(exp) {
		delete(sessions.tokens, c.Value)
		return false
	}
	return true
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
	})
}
```

- [ ] **Step 2: 创建 web/handler.go**

```go
package web

import (
	"embed"
	"html/template"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"

	"agent-mail/service"
	"agent-mail/store/sqlite"
)

//go:embed templates/*
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type AdminHandler struct {
	userSvc *service.UserService
	db      *sqlite.DB
}

func NewAdminHandler(db *sqlite.DB, userSvc *service.UserService) *AdminHandler {
	return &AdminHandler{userSvc: userSvc, db: db}
}

func (h *AdminHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/admin/login", h.handleLogin)
	mux.HandleFunc("/admin/users", h.authWrap(h.handleUsers))
	mux.HandleFunc("/admin/users/create", h.authWrap(h.handleUserCreate))
	mux.HandleFunc("/admin/users/refresh", h.authWrap(h.handleTokenRefresh))
	mux.HandleFunc("/admin/users/delete", h.authWrap(h.handleUserDelete))
	mux.HandleFunc("/admin/logout", h.handleLogout)
	mux.HandleFunc("/admin/", h.handleIndex)
}

func (h *AdminHandler) authWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkSession(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		fn(w, r)
	}
}

func (h *AdminHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/" && r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}
	if checkSession(r) {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

func (h *AdminHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.ExecuteTemplate(w, "login.html", nil)
		return
	}
	password := r.FormValue("password")
	storedHash, _ := h.db.GetSetting("admin_password_hash")
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		tmpl.ExecuteTemplate(w, "login.html", map[string]string{"Error": "密码错误"})
		return
	}
	setSession(w)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	users, _ := h.userSvc.ListUsers()
	type userWithToken struct {
		ID          int
		Name        string
		CreatedAt   string
		TokenPrefix string
	}
	var data []userWithToken
	for _, u := range users {
		data = append(data, userWithToken{
			ID:          u.ID,
			Name:        u.Name,
			CreatedAt:   u.CreatedAt,
			TokenPrefix: h.userSvc.GetActiveTokenPrefix(u.ID),
		})
	}
	tmpl.ExecuteTemplate(w, "users.html", data)
}

func (h *AdminHandler) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.ExecuteTemplate(w, "user_create.html", nil)
		return
	}
	name := r.FormValue("name")
	u, token, err := h.userSvc.CreateUser(name)
	if err != nil {
		tmpl.ExecuteTemplate(w, "user_create.html", map[string]string{"Error": err.Error()})
		return
	}
	tmpl.ExecuteTemplate(w, "user_create.html", map[string]any{
		"Success": true,
		"Name":    u.Name,
		"Token":   token,
	})
}

func (h *AdminHandler) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	// POST only, redirect back
	// Simplified: refresh via query param user_id
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := 0
	fmt.Sscanf(r.FormValue("user_id"), "%d", &userID)
	if userID == 0 {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}
	token, err := h.userSvc.RefreshToken(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "user_detail.html", map[string]any{
		"NewToken": token,
	})
}

func (h *AdminHandler) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := 0
	fmt.Sscanf(r.FormValue("user_id"), "%d", &userID)
	if err := h.userSvc.DeleteUser(userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
```

Add `"fmt"` import.

- [ ] **Step 3: 创建 HTML 模板**

**web/templates/base.html:**
```html
{{define "base"}}<!DOCTYPE html>
<html lang="zh">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{block "title" .}}agent-mail{{end}}</title>
<style>*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#f5f5f5;color:#333}
header{background:#1a1a2e;color:white;padding:1rem;display:flex;justify-content:space-between;align-items:center}
header a{color:#e0e0ff;text-decoration:none}
.container{max-width:900px;margin:2rem auto;padding:0 1rem}
.card{background:white;border-radius:8px;padding:1.5rem;margin-bottom:1rem;box-shadow:0 1px 3px rgba(0,0,0,.1)}
table{width:100%;border-collapse:collapse}
th,td{padding:.75rem;text-align:left;border-bottom:1px solid #eee}
th{font-weight:600;color:#666}
.btn{display:inline-block;padding:.5rem 1rem;border-radius:4px;text-decoration:none;border:none;cursor:pointer;font-size:1rem}
.btn-primary{background:#1a1a2e;color:white}
.btn-danger{background:#dc3545;color:white}
.btn-success{background:#28a745;color:white}
input,textarea{width:100%;padding:.5rem;border:1px solid #ddd;border-radius:4px;font-size:1rem;margin:.25rem 0 1rem}
.form-group{margin-bottom:1rem}
.form-group label{display:block;margin-bottom:.25rem;font-weight:600}
.alert{padding:1rem;border-radius:4px;margin-bottom:1rem}
.alert-error{background:#f8d7da;color:#721c24;border:1px solid #f5c6cb}
.alert-success{background:#d4edda;color:#155724;border:1px solid #c3e6cb}
.token-box{background:#1a1a2e;color:#0f0;padding:1rem;border-radius:4px;font-family:monospace;word-break:break-all;margin:1rem 0}
</style></head>
<body><header>
<a href="/admin/users">agent-mail</a>
<div><a href="/admin/logout">登出</a></div>
</header>
<div class="container">{{template "content" .}}</div>
</body></html>{{end}}
```

**web/templates/login.html:**
```html
{{template "base" .}}
{{define "content"}}
<div class="card" style="max-width:400px;margin:4rem auto">
<h2 style="margin-bottom:1rem">管理员登录</h2>
{{if .Error}}<div class="alert alert-error">{{.Error}}</div>{{end}}
<form method="POST">
<input type="password" name="password" placeholder="管理员密码" required>
<button class="btn btn-primary" style="width:100%">登录</button>
</form>
</div>
{{end}}
```

**web/templates/users.html:**
```html
{{template "base" .}}
{{define "title"}}用户管理{{end}}
{{define "content"}}
<h2 style="margin-bottom:1rem">用户列表</h2>
<div style="margin-bottom:1rem">
<a href="/admin/users/create" class="btn btn-primary">创建用户</a>
</div>
<div class="card">
<table>
<thead><tr><th>ID</th><th>用户名</th><th>Token</th><th>操作</th></tr></thead>
<tbody>
{{range .}}
<tr>
<td>{{.ID}}</td><td>{{.Name}}</td>
<td><code>{{.TokenPrefix}}</code></td>
<td>
<form method="POST" action="/admin/users/refresh" style="display:inline">
<input type="hidden" name="user_id" value="{{.ID}}">
<button class="btn btn-success" style="padding:.25rem .5rem;font-size:.85rem">刷新</button>
</form>
<form method="POST" action="/admin/users/delete" style="display:inline" onsubmit="return confirm('确认删除用户 {{.Name}}？')">
<input type="hidden" name="user_id" value="{{.ID}}">
<button class="btn btn-danger" style="padding:.25rem .5rem;font-size:.85rem">删除</button>
</form>
</td>
</tr>
{{end}}
{{if not .}}
<tr><td colspan="4" style="text-align:center;color:#999">暂无用户</td></tr>
{{end}}
</tbody>
</table>
</div>
{{if .NewToken}}
<div class="card">
<h3>新 Token（仅显示一次！）</h3>
<div class="token-box">{{.NewToken}}</div>
<a href="/admin/users" class="btn btn-primary">返回用户列表</a>
</div>
{{end}}
{{end}}
```

**web/templates/user_create.html:**
```html
{{template "base" .}}
{{define "title"}}创建用户{{end}}
{{define "content"}}
{{if .Success}}
<div class="card">
<h2 style="margin-bottom:1rem">用户创建成功</h2>
<div class="alert alert-success">用户 <strong>{{.Name}}</strong> 已创建</div>
<h3 style="margin-top:1rem">Token（仅显示一次！请立即复制）</h3>
<div class="token-box">{{.Token}}</div>
<p style="margin-top:1rem;color:#666">此 Token 注销后将无法再次查看，请妥善保管。</p>
<p>MCP 客户端配置示例：</p>
<div class="token-box" style="color:#aaa">
{"headers":{"X-Agent-Mail-Token":"{{.Token}}"}}
</div>
<a href="/admin/users" class="btn btn-primary" style="margin-top:1rem">返回用户列表</a>
</div>
{{else}}
<div class="card" style="max-width:400px;margin:2rem auto">
<h2 style="margin-bottom:1rem">创建新用户</h2>
{{if .Error}}<div class="alert alert-error">{{.Error}}</div>{{end}}
<form method="POST">
<div class="form-group">
<label>用户名</label>
<input type="text" name="name" required placeholder="输入用户名">
</div>
<button class="btn btn-primary" style="width:100%">创建</button>
</form>
<a href="/admin/users" style="display:block;margin-top:1rem;text-align:center;color:#666">返回</a>
</div>
{{end}}
{{end}}
```

**web/templates/user_detail.html:** (token 刷新结果页面，已嵌入 users.html 中的 NewToken 逻辑)

- [ ] **Step 4: 编译验证**

```bash
go build ./web/...
```

- [ ] **Step 5: Commit**

```bash
git add web/ && git commit -m "feat: add admin web panel with embedded templates"
```

---

### Task 9: main.go — 挂载 admin 路由，初始化 admin 密码

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 更新 main.go**

```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"agent-mail/mcp"
	"agent-mail/service"
	"agent-mail/store/sqlite"
	"agent-mail/web"
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

	// Admin password
	initAdminPassword(db)

	// Services
	userSvc := service.NewUserService(db)
	mailboxSvc := service.NewMailboxService(db, nil)
	emailSvc := service.NewEmailService(mailboxSvc)
	sendSvc := service.NewSendService(mailboxSvc)
	autoReplySvc := service.NewAutoReplyService(mailboxSvc)
	webhookSvc := service.NewWebhookService(mailboxSvc)
	attachmentSvc := service.NewAttachmentService(mailboxSvc)

	// MCP
	handler := mcp.NewHandler(mailboxSvc, emailSvc, sendSvc, autoReplySvc, webhookSvc, attachmentSvc)
	srv := mcp.NewServer(*addr, handler, userSvc)

	// Admin web
	adminHandler := web.NewAdminHandler(db, userSvc)
	adminHandler.Register(srv.Mux())

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
```

同时需要在 `mcp/server.go` 中暴露 `Mux()` 方法：

```go
func (s *Server) Mux() *http.ServeMux {
	// 需要在 Start 之前也能拿到 mux，所以改为在 NewServer 中创建 mux
}
```

重构 mcp/server.go：把 mux 创建和路由注册移到 `NewServer` 或加一个 `InitRoutes`：

```go
type Server struct {
	httpServer *http.Server
	handler    *Handler
	userSvc    *service.UserService
	mux        *http.ServeMux
}

func NewServer(addr string, handler *Handler, userSvc *service.UserService) *Server {
	mcpSrv := goserver.NewMCPServer("agent-mail", "1.0.0")
	for _, tool := range Tools {
		mcpSrv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handler.HandleToolCall(ctx, req)
		})
	}
	streamableServer := goserver.NewStreamableHTTPServer(mcpSrv)

	mux := http.NewServeMux()
	mux.Handle("/mcp", authMiddleware(streamableServer, userSvc))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		handler: handler,
		userSvc: userSvc,
		mux:     mux,
	}
}

func (s *Server) Mux() *http.ServeMux { return s.mux }

func (s *Server) Start() error {
	slog.Info("MCP server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}
```

- [ ] **Step 2: 编译 + 运行测试**

```bash
go build . && go test ./... -count=1
```

- [ ] **Step 3: Commit**

```bash
git add main.go mcp/server.go && git commit -m "feat: mount admin panel, auto-init admin password"
```

---

### Task 10: 更新现有测试 + 最终验证

**Files:**
- Modify: `service/mailbox_svc_test.go` (所有调用加 userID)
- Modify: `mcp/tools_test.go` (setupHandler 加 userID 到 context)

- [ ] **Step 1: 更新 service/mailbox_svc_test.go**

所有 `svc.Add(...)` → `svc.Add(1, ...)`，所有 `svc.List()` → `svc.List(1)`，所有 `svc.Switch(...)` → `svc.Switch(1, ...)`，所有 `svc.Resolve(...)` → `svc.Resolve(1, ...)`。

- [ ] **Step 2: 更新 mcp/tools_test.go**

```go
func setupHandler(t *testing.T) *Handler {
	// ...same as before, but Add 调用改为:
	ms.Add(1, "test", "Test", "cloudflare", "https://mail.example.com", `{"jwt":"tok","site_password":""}`)
	// 所有 handler 测试调用需要传入带 user_id 的 context:
	// ctx := context.WithValue(t.Context(), UserIDKey, 1)
}

// 更新所有 TestHandler* 函数，ctx 改为 context.WithValue(t.Context(), UserIDKey, 1)
```

- [ ] **Step 3: 运行全部测试**

```bash
go test ./... -count=1 -v
```

- [ ] **Step 4: 最终验证**

```bash
go vet ./... && go build -o agent-mail .
```

- [ ] **Step 5: Commit**

```bash
git add . && git commit -m "test: update existing tests for multi-user isolation"
```
