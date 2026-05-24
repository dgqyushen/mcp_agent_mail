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

type testProviderFactory struct{}

func (f *testProviderFactory) NewProvider(record model.MailboxRecord) (provider.EmailProvider, error) {
	auth := make(map[string]string)
	json.Unmarshal([]byte(record.AuthData), &auth)
	return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"]), nil
}

func TestMailboxServiceAddList(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewMailboxService(db, &testProviderFactory{})

	err = svc.Add(1, "work", "Work", "cloudflare", "https://mail.example.com", `{"jwt":"token123"}`)
	if err != nil {
		t.Fatal(err)
	}

	list, err := svc.List(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 mailbox, got %d", len(list))
	}
	if list[0].Alias != "work" {
		t.Errorf("expected alias work, got %q", list[0].Alias)
	}
}

func TestMailboxServiceSwitchDefault(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewMailboxService(db, &testProviderFactory{})

	svc.Add(1, "a", "A", "cloudflare", "https://a.com", "{}")
	svc.Add(1, "b", "B", "cloudflare", "https://b.com", "{}")

	if def := svc.Default(1); def != "a" {
		t.Errorf("expected default a, got %q", def)
	}

	if err := svc.Switch(1, "b"); err != nil {
		t.Fatal(err)
	}
	if def := svc.Default(1); def != "b" {
		t.Errorf("expected default b after switch, got %q", def)
	}
}

func TestMailboxServiceRemove(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewMailboxService(db, &testProviderFactory{})

	svc.Add(1, "a", "A", "cloudflare", "https://a.com", "{}")
	svc.Add(1, "b", "B", "cloudflare", "https://b.com", "{}")

	if err := svc.Remove(1, "a"); err != nil {
		t.Fatal(err)
	}
	// Default should fallback to remaining one
	if def := svc.Default(1); def != "b" {
		t.Errorf("expected default b after removing a, got %q", def)
	}

	list, _ := svc.List(1)
	if len(list) != 1 {
		t.Errorf("expected 1 mailbox after remove, got %d", len(list))
	}
}

func TestMailboxServiceResolve(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewMailboxService(db, &testProviderFactory{})

	svc.Add(1, "work", "Work", "cloudflare", "https://mail.example.com", `{"jwt":"token123"}`)

	rec, err := svc.Resolve(1, "work")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Alias != "work" {
		t.Errorf("expected work, got %q", rec.Alias)
	}

	// Resolve with empty should use default
	rec, err = svc.Resolve(1, "")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Alias != "work" {
		t.Errorf("expected default work, got %q", rec.Alias)
	}
}
