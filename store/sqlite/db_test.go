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

	// Also verify settings table exists
	err = db.Conn().QueryRow("SELECT COUNT(*) FROM settings").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
}
