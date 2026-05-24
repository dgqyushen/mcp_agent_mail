package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type userSessionStore struct {
	mu     sync.Mutex
	tokens map[string]time.Time
}

var userSessions = &userSessionStore{tokens: make(map[string]time.Time)}

func newUserSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

const userSessionCookie = "user_session"
const userSessionTTL = 12 * time.Hour

func setUserSession(w http.ResponseWriter) string {
	sid := newUserSession()
	userSessions.mu.Lock()
	userSessions.tokens[sid] = time.Now().Add(userSessionTTL)
	userSessions.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     userSessionCookie,
		Value:    sid,
		Path:     "/user",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(userSessionTTL.Seconds()),
	})
	return sid
}

func checkUserSession(r *http.Request) bool {
	c, err := r.Cookie(userSessionCookie)
	if err != nil {
		return false
	}
	userSessions.mu.Lock()
	defer userSessions.mu.Unlock()
	exp, ok := userSessions.tokens[c.Value]
	if !ok || time.Now().After(exp) {
		delete(userSessions.tokens, c.Value)
		return false
	}
	return true
}

func clearUserSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   userSessionCookie,
		Value:  "",
		Path:   "/user",
		MaxAge: -1,
	})
}
