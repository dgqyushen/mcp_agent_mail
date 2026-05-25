package mcp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"agent-mail/service"
	"agent-mail/store/sqlite"
)

func setupAuthTest(t *testing.T) (*service.UserService, func()) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() { db.Close() }
	userSvc := service.NewUserService(db)
	_, _, err = userSvc.CreateUser("testuser")
	if err != nil {
		t.Fatal(err)
	}
	return userSvc, cleanup
}

func TestAuthMiddlewareMissingToken(t *testing.T) {
	userSvc, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := AuthMiddleware(handler, userSvc)

	req := httptest.NewRequest("GET", "/mcp", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareMissingTokenPost(t *testing.T) {
	userSvc, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := AuthMiddleware(handler, userSvc)

	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareMissingTokenDelete(t *testing.T) {
	userSvc, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := AuthMiddleware(handler, userSvc)

	req := httptest.NewRequest("DELETE", "/mcp", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	userSvc, cleanup := setupAuthTest(t)
	defer cleanup()

	user, token, err := userSvc.CreateUser("authtest")
	if err != nil {
		t.Fatal(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID != user.ID {
			t.Errorf("expected userID %d, got %d", user.ID, userID)
		}
		w.WriteHeader(http.StatusOK)
	})
	mw := AuthMiddleware(handler, userSvc)

	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("X-Agent-Mail-Token", token)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	userSvc, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := AuthMiddleware(handler, userSvc)

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-Agent-Mail-Token", "invalid-token-12345")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareLegacyToken(t *testing.T) {
	os.Setenv("AUTH_HEADER", "X-Custom-Auth")
	os.Setenv("AUTH_TOKEN", "legacy-secret")
	defer func() {
		os.Unsetenv("AUTH_HEADER")
		os.Unsetenv("AUTH_TOKEN")
	}()

	userSvc, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID != 0 {
			t.Errorf("expected legacy userID 0, got %d", userID)
		}
		w.WriteHeader(http.StatusOK)
	})
	mw := AuthMiddleware(handler, userSvc)

	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("X-Custom-Auth", "legacy-secret")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
