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

	if _, err := db.CreateUser("TestUser"); err != nil {
		t.Fatal(err)
	}

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
}

func TestTokenDeactivate(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.CreateUser("TestUser"); err != nil {
		t.Fatal(err)
	}
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
