package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type userSessionData struct {
	userID int
	expiry time.Time
}

type userSessionStore struct {
	mu     sync.Mutex
	tokens map[string]userSessionData
}

var userSessions = &userSessionStore{tokens: make(map[string]userSessionData)}

func newUserSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

const userSessionCookie = "user_session"
const userSessionTTL = 12 * time.Hour

func setUserSession(w http.ResponseWriter, userID int) string {
	sid := newUserSession()
	userSessions.mu.Lock()
	userSessions.tokens[sid] = userSessionData{userID: userID, expiry: time.Now().Add(userSessionTTL)}
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
	data, ok := userSessions.tokens[c.Value]
	if !ok || time.Now().After(data.expiry) {
		delete(userSessions.tokens, c.Value)
		return false
	}
	return true
}

func getUserID(r *http.Request) int {
	c, err := r.Cookie(userSessionCookie)
	if err != nil {
		return 0
	}
	userSessions.mu.Lock()
	defer userSessions.mu.Unlock()
	data, ok := userSessions.tokens[c.Value]
	if !ok || time.Now().After(data.expiry) {
		delete(userSessions.tokens, c.Value)
		return 0
	}
	return data.userID
}

func clearUserSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   userSessionCookie,
		Value:  "",
		Path:   "/user",
		MaxAge: -1,
	})
}
