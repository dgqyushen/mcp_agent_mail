package mcp

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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
