package web

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type sessionStore struct {
	mu     sync.Mutex
	tokens map[string]time.Time
}

var sessions = &sessionStore{tokens: make(map[string]time.Time)}

func newSession() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		slog.Error("rand.Read failed generating session", "error", err)
	}
	return hex.EncodeToString(b)
}

const sessionCookie = "admin_session"
const sessionTTL = 12 * time.Hour

func setSession(w http.ResponseWriter) string {
	sid := newSession()
	sessions.mu.Lock()
	sessions.tokens[sid] = time.Now().Add(sessionTTL)
	sessions.mu.Unlock()
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
	sessions.mu.Lock()
	defer sessions.mu.Unlock()
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

func (s *sessionStore) startCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.mu.Lock()
			now := time.Now()
			for id, exp := range s.tokens {
				if now.After(exp) {
					delete(s.tokens, id)
				}
			}
			s.mu.Unlock()
		}
	}()
}

func StartSessionCleanup(interval time.Duration) {
	sessions.startCleanup(interval)
}
