package sqlite_test

import (
	"testing"

	"agent-mail/store/sqlite"
)

func TestSettingsGetSet(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SetSetting("key1", "val1"); err != nil {
		t.Fatal(err)
	}

	v, err := db.GetSetting("key1")
	if err != nil {
		t.Fatal(err)
	}
	if v != "val1" {
		t.Errorf("expected val1, got %q", v)
	}
}

func TestSettingsGetDefault(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	v, err := db.GetSetting("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}
