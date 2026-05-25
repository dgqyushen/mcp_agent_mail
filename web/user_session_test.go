package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSetAndCheckUserSession(t *testing.T) {
	w := httptest.NewRecorder()
	sid := setUserSession(w, 0)
	if sid == "" {
		t.Fatal("expected non-empty session ID")
	}
	resp := w.Result()
	cookies := resp.Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == userSessionCookie {
			found = true
			if c.Path != "/user" {
				t.Errorf("expected path /user, got %s", c.Path)
			}
			if !c.HttpOnly {
				t.Error("expected HttpOnly")
			}
			break
		}
	}
	if !found {
		t.Fatal("expected user_session cookie")
	}
	req := httptest.NewRequest("GET", "/user/mailboxes", nil)
	req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: sid})
	if !checkUserSession(req) {
		t.Error("expected valid session")
	}
}

func TestUserSessionCleanup(t *testing.T) {
	store := &userSessionStore{tokens: make(map[string]userSessionData)}

	store.mu.Lock()
	store.tokens["expired"] = userSessionData{userID: 1, expiry: time.Now().Add(-1 * time.Hour)}
	store.tokens["valid"] = userSessionData{userID: 2, expiry: time.Now().Add(1 * time.Hour)}
	store.mu.Unlock()

	store.startCleanup(50 * time.Millisecond)

	time.Sleep(150 * time.Millisecond)

	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.tokens["expired"]; ok {
		t.Error("expired user session should have been cleaned up")
	}
	if _, ok := store.tokens["valid"]; !ok {
		t.Error("valid user session should not have been cleaned up")
	}
}
