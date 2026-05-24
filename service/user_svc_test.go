package service_test

import (
	"testing"

	"agent-mail/service"
	"agent-mail/store/sqlite"
)

func TestUserServiceCreateAndValidateToken(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)

	u, token, err := svc.CreateUser("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if u.Name != "Alice" {
		t.Errorf("expected Alice, got %q", u.Name)
	}
	if len(token) < 36 || token[:4] != "atm-" {
		t.Errorf("invalid token format: %q", token)
	}

	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if userID != u.ID {
		t.Errorf("expected userID %d, got %d", u.ID, userID)
	}
}

func TestUserServiceRefreshToken(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)

	u, oldToken, err := svc.CreateUser("Bob")
	if err != nil {
		t.Fatal(err)
	}

	newToken, err := svc.RefreshToken(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newToken == oldToken {
		t.Fatal("refresh should produce new token")
	}

	// Old token should be invalid
	_, err = svc.ValidateToken(oldToken)
	if err == nil {
		t.Fatal("old token should be invalid after refresh")
	}

	// New token should work
	uid, err := svc.ValidateToken(newToken)
	if err != nil {
		t.Fatal(err)
	}
	if uid != u.ID {
		t.Errorf("expected userID %d, got %d", u.ID, uid)
	}
}

func TestUserServiceInvalidToken(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)
	_, err = svc.ValidateToken("atm-badtoken")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestUserServiceListAndDelete(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := service.NewUserService(db)
	svc.CreateUser("A")
	svc.CreateUser("B")

	users, err := svc.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	if err := svc.DeleteUser(1); err != nil {
		t.Fatal(err)
	}
	users, _ = svc.ListUsers()
	if len(users) != 1 {
		t.Errorf("expected 1 user after delete, got %d", len(users))
	}
}
