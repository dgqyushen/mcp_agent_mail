package mcp

import (
	"context"
	"net/http"
	"os"

	"agent-mail/service"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func GetUserID(ctx context.Context) int {
	v, _ := ctx.Value(UserIDKey).(int)
	return v
}

func AuthMiddleware(next http.Handler, userSvc *service.UserService) http.Handler {
	legacyHeader := os.Getenv("AUTH_HEADER")
	legacyToken := os.Getenv("AUTH_TOKEN")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Agent-Mail-Token")
		if token == "" && legacyHeader != "" {
			token = r.Header.Get(legacyHeader)
		}

		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized: missing X-Agent-Mail-Token header"}`))
			return
		}

		if legacyToken != "" && token == legacyToken {
			ctx := context.WithValue(r.Context(), UserIDKey, 0)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		userID, err := userSvc.ValidateToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized: invalid or expired token"}`))
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
