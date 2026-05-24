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
