package mcp

import (
	"net/http"
	"os"
)

func authMiddleware(next http.Handler) http.Handler {
	header := os.Getenv("AUTH_HEADER")
	token := os.Getenv("AUTH_TOKEN")

	if header == "" || token == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(header) != token {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
