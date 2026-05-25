package web

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSetAndCheckSession(t *testing.T) {
	w := httptest.NewRecorder()
	sid := setSession(w)
	if sid == "" {
		t.Fatal("expected non-empty session ID")
	}
	resp := w.Result()
	cookies := resp.Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == sessionCookie {
			found = true
			if c.Path != "/admin" {
				t.Errorf("expected path /admin, got %s", c.Path)
			}
			if !c.HttpOnly {
				t.Error("expected HttpOnly")
			}
			break
		}
	}
	if !found {
		t.Fatal("expected admin_session cookie")
	}
	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: sid})
	if !checkSession(req) {
		t.Error("expected valid session")
	}
}

func TestSessionConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	n := 10
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			sid := setSession(w)
			if sid == "" {
				t.Error("expected non-empty session ID")
			}
		}()
	}
	wg.Wait()

	wg2 := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			w := httptest.NewRecorder()
			sid := setSession(w)
			req := httptest.NewRequest("GET", "/admin", nil)
			req.AddCookie(&http.Cookie{Name: sessionCookie, Value: sid})
			_ = checkSession(req)
		}()
	}
	wg2.Wait()
}

func TestSessionCleanup(t *testing.T) {
	store := &sessionStore{tokens: make(map[string]time.Time)}

	store.mu.Lock()
	store.tokens["expired"] = time.Now().Add(-1 * time.Hour)
	store.tokens["valid"] = time.Now().Add(1 * time.Hour)
	store.mu.Unlock()

	store.startCleanup(50 * time.Millisecond)

	time.Sleep(150 * time.Millisecond)

	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.tokens["expired"]; ok {
		t.Error("expired session should have been cleaned up")
	}
	if _, ok := store.tokens["valid"]; !ok {
		t.Error("valid session should not have been cleaned up")
	}
}
