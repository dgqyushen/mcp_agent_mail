package web

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
)

const csrfCookieName = "csrf_token"

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		slog.Error("rand.Read failed generating CSRF token", "error", err)
	}
	return hex.EncodeToString(b)
}

func setCSRFToken(w http.ResponseWriter) string {
	token := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		MaxAge:   86400,
	})
	return token
}

func validateCSRFToken(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return false
	}
	return cookie.Value != "" && cookie.Value == r.FormValue("csrf_token")
}
