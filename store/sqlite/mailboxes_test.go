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

func TestMailboxNotFound(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.GetMailbox("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMailboxDeleteNotFound(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.DeleteMailbox("nonexistent")
	if err == nil {
		t.Fatal("expected error")
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
