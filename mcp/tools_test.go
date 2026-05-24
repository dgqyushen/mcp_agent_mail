package mcp

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"agent-mail/service"
	"agent-mail/store/sqlite"
)

func TestStrArg(t *testing.T) {
	m := map[string]any{"key": "val", "num": 42}
	if got := strArg(m, "key"); got != "val" {
		t.Fatalf("strArg(key) = %q, want %q", got, "val")
	}
	if got := strArg(m, "num"); got != "" {
		t.Fatalf("strArg(num) = %q, want empty", got)
	}
	if got := strArg(m, "missing"); got != "" {
		t.Fatalf("strArg(missing) = %q, want empty", got)
	}
}

func TestIntArg(t *testing.T) {
	m := map[string]any{"a": 42, "b": 3.14, "c": "str"}
	if got := intArg(m, "a"); got != 42 {
		t.Fatalf("intArg(a) = %d, want %d", got, 42)
	}
	if got := intArg(m, "b"); got != 3 {
		t.Fatalf("intArg(b) = %d, want %d", got, 3)
	}
	if got := intArg(m, "c"); got != 0 {
		t.Fatalf("intArg(c) = %d, want 0", got)
	}
	if got := intArg(m, "missing"); got != 0 {
		t.Fatalf("intArg(missing) = %d, want 0", got)
	}
}

func TestBoolArg(t *testing.T) {
	m := map[string]any{"t": true, "f": false, "s": "true"}
	if got := boolArg(m, "t"); got != true {
		t.Fatalf("boolArg(t) = %v, want true", got)
	}
	if got := boolArg(m, "f"); got != false {
		t.Fatalf("boolArg(f) = %v, want false", got)
	}
	if got := boolArg(m, "s"); got != false {
		t.Fatalf("boolArg(s) = %v, want false", got)
	}
	if got := boolArg(m, "missing"); got != false {
		t.Fatalf("boolArg(missing) = %v, want false", got)
	}
}

func TestToResult(t *testing.T) {
	r := toResult(map[string]string{"k": "v"})
	if r.IsError {
		t.Fatal("toResult should not set IsError")
	}
	if len(r.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(r.Content))
	}
	tc, ok := r.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(tc.Text), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["k"] != "v" {
		t.Fatalf("unexpected value: %v", decoded)
	}
}

func TestErrorResult(t *testing.T) {
	r := errorResult(errors.New("test error"))
	if !r.IsError {
		t.Fatal("errorResult should set IsError")
	}
	if len(r.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(r.Content))
	}
	tc, ok := r.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text == "" {
		t.Fatal("error message should not be empty")
	}
}

func setupHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ms := service.NewMailboxService(db, nil)
	ms.Add("test", "Test", "cloudflare", "https://mail.example.com", `{"jwt":"tok","site_password":""}`)
	return NewHandler(
		ms,
		service.NewEmailService(ms),
		service.NewSendService(ms),
		service.NewAutoReplyService(ms),
		service.NewWebhookService(ms),
		service.NewAttachmentService(ms),
	)
}

func TestHandlerListMailboxes(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolListMailboxes,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %v", res.Content[0].(mcp.TextContent).Text)
	}
	var list []map[string]any
	json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 mailbox, got %d", len(list))
	}
}

func TestHandlerAddMailbox(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: ToolAddMailbox,
			Arguments: map[string]any{
				"alias":         "work",
				"name":          "Work",
				"provider_type": "cloudflare",
				"base_url":      "https://work.example.com",
				"auth_data":     `{"jwt":"worktok"}`,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %v", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestHandlerRemoveMailbox(t *testing.T) {
	h := setupHandler(t)
	_, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolRemoveMailbox,
			Arguments: map[string]any{"alias": "test"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerRemoveMailboxMissingAlias(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolRemoveMailbox,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing alias")
	}
}

func TestHandlerSwitchMailbox(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSwitchMailbox,
			Arguments: map[string]any{"alias": "test"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %v", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestHandlerSwitchMailboxNotFound(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSwitchMailbox,
			Arguments: map[string]any{"alias": "nope"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for nonexistent mailbox")
	}
}

func TestHandlerDeleteEmailMissingID(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolDeleteEmail,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing id parameter")
	}
}

func TestHandlerDeleteSentMissingID(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolDeleteSent,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing id parameter")
	}
}

func TestHandlerSendMailMissingRequired(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSendMail,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing required parameters")
	}
}

func TestHandlerSetAutoReplyMissingEnabled(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSetAutoReply,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing enabled parameter")
	}
}

func TestHandlerSetWebhookMissingRequired(t *testing.T) {
	h := setupHandler(t)
	res, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSetWebhook,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing required parameters")
	}
}

func TestHandlerUnknownTool(t *testing.T) {
	h := setupHandler(t)
	_, err := h.HandleToolCall(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "nonexistent_tool",
			Arguments: map[string]any{},
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}
