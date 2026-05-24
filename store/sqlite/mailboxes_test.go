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
		UserID:       1,
		Alias:        "work",
		Name:         "Work",
		ProviderType: "cloudflare",
		BaseURL:      "https://mail.example.com",
		AuthData:     `{"jwt":"token123"}`,
	}

	if err := db.InsertMailbox(rec); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetMailbox(1, "work")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Work" {
		t.Errorf("expected Work, got %q", got.Name)
	}

	list, err := db.ListMailboxes(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 mailbox, got %d", len(list))
	}

	if err := db.DeleteMailbox(1, "work"); err != nil {
		t.Fatal(err)
	}

	list, err = db.ListMailboxes(1)
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

	_, err = db.GetMailbox(1, "nonexistent")
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

	err = db.DeleteMailbox(1, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMailboxUserIsolation(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.CreateUser("Alice")
	db.CreateUser("Bob")

	db.InsertMailbox(model.MailboxRecord{UserID: 1, Alias: "inbox", Name: "AInbox", BaseURL: "https://a.com", AuthData: "{}"})
	db.InsertMailbox(model.MailboxRecord{UserID: 2, Alias: "inbox", Name: "BInbox", BaseURL: "https://b.com", AuthData: "{}"})

	aliceList, _ := db.ListMailboxes(1)
	if len(aliceList) != 1 || aliceList[0].Name != "AInbox" {
		t.Errorf("Alice should see 1 mailbox, got %+v", aliceList)
	}

	bobList, _ := db.ListMailboxes(2)
	if len(bobList) != 1 || bobList[0].Name != "BInbox" {
		t.Errorf("Bob should see 1 mailbox, got %+v", bobList)
	}

	_, err = db.GetMailbox(1, "inbox")
	if err != nil {
		t.Errorf("Alice should get her own inbox, got error: %v", err)
	}

	err = db.DeleteMailbox(1, "inbox")
	if err != nil {
		t.Errorf("Alice should delete her own inbox, got: %v", err)
	}
}

func TestUpdateMailbox(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rec := model.MailboxRecord{
		UserID:       1,
		Alias:        "work",
		Name:         "Work",
		ProviderType: "cloudflare",
		BaseURL:      "https://old.example.com",
		AuthData:     `{"jwt":"old"}`,
	}
	if err := db.InsertMailbox(rec); err != nil {
		t.Fatal(err)
	}

	updated := model.MailboxRecord{
		UserID:       1,
		Alias:        "work",
		Name:         "Work Updated",
		ProviderType: "gmail",
		BaseURL:      "https://new.example.com",
		AuthData:     `{"token":"new"}`,
	}
	if err := db.UpdateMailbox(updated); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetMailbox(1, "work")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Work Updated" {
		t.Errorf("expected 'Work Updated', got %q", got.Name)
	}
	if got.ProviderType != "gmail" {
		t.Errorf("expected gmail, got %q", got.ProviderType)
	}
	if got.BaseURL != "https://new.example.com" {
		t.Errorf("expected new base URL, got %q", got.BaseURL)
	}
}

func TestMailboxDuplicate(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rec := model.MailboxRecord{UserID: 1, Alias: "x", Name: "X", BaseURL: "https://x.com", AuthData: "{}"}
	if err := db.InsertMailbox(rec); err != nil {
		t.Fatal(err)
	}
	err = db.InsertMailbox(rec)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}
