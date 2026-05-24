package web

import (
	"net/http"

	"agent-mail/service"
)

type UserHandler struct {
	userSvc *service.UserService
}

func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/user/login", h.handleLogin)
	mux.HandleFunc("/user/logout", h.handleLogout)
}

func (h *UserHandler) authWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkUserSession(r) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}
		fn(w, r)
	}
}

func (h *UserHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		pages["user_login"].ExecuteTemplate(w, "base", nil)
		return
	}
	token := r.FormValue("token")
	_, err := h.userSvc.ValidateToken(token)
	if err != nil {
		pages["user_login"].ExecuteTemplate(w, "base", map[string]string{"Error": "Token 无效或已过期"})
		return
	}
	setUserSession(w)
	http.Redirect(w, r, "/user/mailboxes", http.StatusSeeOther)
}

func (h *UserHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearUserSession(w)
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}
