package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetAndCheckUserSession(t *testing.T) {
	w := httptest.NewRecorder()
	sid := setUserSession(w)
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
