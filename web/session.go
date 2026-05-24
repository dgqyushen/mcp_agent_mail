package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

type sessionStore struct {
	tokens map[string]time.Time
}

var sessions = &sessionStore{tokens: make(map[string]time.Time)}

func newSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

const sessionCookie = "admin_session"
const sessionTTL = 12 * time.Hour

func setSession(w http.ResponseWriter) string {
	sid := newSession()
	sessions.tokens[sid] = time.Now().Add(sessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sid,
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return sid
}

func checkSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	exp, ok := sessions.tokens[c.Value]
	if !ok || time.Now().After(exp) {
		delete(sessions.tokens, c.Value)
		return false
	}
	return true
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookie,
		Value:  "",
		Path:   "/admin",
		MaxAge: -1,
	})
}
