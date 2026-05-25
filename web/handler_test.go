package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminHandler_AuthWrap_RedirectsWhenNotLoggedIn(t *testing.T) {
	h := &AdminHandler{}
	req := httptest.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()

	handler := h.authWrap(h.handleUsers)
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("expected redirect 303, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/admin/" {
		t.Errorf("expected Location /admin/, got %s", loc)
	}
}

func TestAdminHandler_AuthWrap_AllowsWhenLoggedIn(t *testing.T) {
	h := &AdminHandler{}
	sidW := httptest.NewRecorder()
	sid := setSession(sidW)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: sid})
	w := httptest.NewRecorder()

	handler := h.authWrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Error("expected 200 OK for authenticated request")
	}
}
