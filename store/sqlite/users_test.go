package sqlite_test

import (
	"testing"

	"agent-mail/store/sqlite"
)

func TestUsersCRUD(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	u, err := db.CreateUser("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 || u.Name != "Alice" {
		t.Errorf("unexpected user: %+v", u)
	}

	got, err := db.GetUser(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Alice" {
		t.Errorf("expected Alice, got %q", got.Name)
	}

	list, err := db.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 user, got %d", len(list))
	}
}

func TestUserNotFound(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.GetUser(999)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteUser(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.CreateUser("Bob")
	if err := db.DeleteUser(1); err != nil {
		t.Fatal(err)
	}
	_, err = db.GetUser(1)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}
