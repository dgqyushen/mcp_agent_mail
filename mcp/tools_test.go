package mcp

import (
	"context"
	"encoding/json"
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

func TestHandleListMailboxes_NoAPI(t *testing.T) {
	s, _ := newTestServer(t)
	result, err := s.handleListMailboxes(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected no error from list_mailboxes")
	}
	var infos []model.MailboxInfo
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &infos); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 mailbox, got %d", len(infos))
	}
	if infos[0].Alias != "test" || infos[0].Name != "Test" {
		t.Fatalf("unexpected mailbox: %+v", infos[0])
	}
}

func TestHandleValidateMailbox_NotFound(t *testing.T) {
	s, _ := newTestServer(t)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"alias": "nonexistent"}
	result, _ := s.handleValidateMailbox(context.Background(), req)
	if !result.IsError {
		t.Fatal("expected error for nonexistent mailbox")
	}
}

func TestHandleAddMailbox_MissingRequired(t *testing.T) {
	s, _ := newTestServer(t)
	result, _ := s.handleAddMailbox(context.Background(), mcp.CallToolRequest{})
	if !result.IsError {
		t.Fatal("expected error for missing required params")
	}
}

func TestHandleRemoveMailbox_NotFound(t *testing.T) {
	s, _ := newTestServer(t)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"alias": "nonexistent"}
	result, _ := s.handleRemoveMailbox(context.Background(), req)
	if !result.IsError {
		t.Fatal("expected error for nonexistent mailbox")
	}
}

func TestHandleSwitchMailbox_NotFound(t *testing.T) {
	s, _ := newTestServer(t)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"alias": "nonexistent"}
	result, _ := s.handleSwitchMailbox(context.Background(), req)
	if !result.IsError {
		t.Fatal("expected error for nonexistent mailbox")
	}
}

func TestToJSON(t *testing.T) {
	result := toJSON(map[string]string{"key": "value"})
	if result != `{"key":"value"}` {
		t.Fatalf("unexpected JSON: %s", result)
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		raw, def string
		min, max int
		want     int
		wantErr  bool
	}{
		{"", "10", 1, 100, 10, false},
		{"5", "10", 1, 100, 5, false},
		{"0", "10", 1, 100, 0, true},
		{"101", "10", 1, 100, 0, true},
		{"abc", "10", 1, 100, 0, true},
	}
	for _, tt := range tests {
		got, err := parseIntParam(tt.raw, tt.def, tt.min, tt.max)
		if tt.wantErr && err == nil {
			t.Errorf("parseIntParam(%q, %q, %d, %d) expected error", tt.raw, tt.def, tt.min, tt.max)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("parseIntParam(%q, %q, %d, %d) unexpected error: %v", tt.raw, tt.def, tt.min, tt.max, err)
		}
		if got != tt.want {
			t.Errorf("parseIntParam(%q, %q, %d, %d) = %d, want %d", tt.raw, tt.def, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestParseMailID(t *testing.T) {
	tests := []struct {
		raw     interface{}
		want    int
		wantErr bool
	}{
		{"123", 123, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{"mail_id": tt.raw}
		got, err := parseMailID(req)
		if tt.wantErr && err == nil {
			t.Errorf("parseMailID(%v) expected error", tt.raw)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("parseMailID(%v) unexpected error: %v", tt.raw, err)
		}
		if got != tt.want {
			t.Errorf("parseMailID(%v) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}
