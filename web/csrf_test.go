package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateCSRFToken(t *testing.T) {
	t1 := generateCSRFToken()
	t2 := generateCSRFToken()
	if t1 == "" || t2 == "" {
		t.Fatal("empty token")
	}
	if t1 == t2 {
		t.Fatal("tokens should be unique")
	}
	if len(t1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(t1))
	}
}

func TestSetCSRFToken(t *testing.T) {
	w := httptest.NewRecorder()
	token := setCSRFToken(w)
	if token == "" {
		t.Fatal("empty token")
	}

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == csrfCookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("csrf cookie not set")
	}
	if found.Value != token {
		t.Fatalf("cookie value %q != token %q", found.Value, token)
	}
	if !found.HttpOnly {
		t.Fatal("csrf cookie not HttpOnly")
	}
	if found.SameSite != http.SameSiteStrictMode {
		t.Fatal("csrf cookie not SameSite=Strict")
	}
}

func TestValidateCSRFToken(t *testing.T) {
	token := "abc123"

	t.Run("valid", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", nil)
		r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
		r.PostForm = map[string][]string{"csrf_token": {token}}
		if !validateCSRFToken(r) {
			t.Fatal("should be valid")
		}
	})

	t.Run("no cookie", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", nil)
		r.PostForm = map[string][]string{"csrf_token": {token}}
		if validateCSRFToken(r) {
			t.Fatal("should be invalid without cookie")
		}
	})

	t.Run("no form value", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", nil)
		r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
		if validateCSRFToken(r) {
			t.Fatal("should be invalid without form value")
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", nil)
		r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
		r.PostForm = map[string][]string{"csrf_token": {"wrong"}}
		if validateCSRFToken(r) {
			t.Fatal("should be invalid on mismatch")
		}
	})

	t.Run("empty cookie value", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", nil)
		r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: ""})
		r.PostForm = map[string][]string{"csrf_token": {token}}
		if validateCSRFToken(r) {
			t.Fatal("should be invalid with empty cookie")
		}
	})
}
