package web

import (
	"embed"
	"html/template"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"agent-mail/service"
	"agent-mail/store/sqlite"
)

//go:embed templates/base.html templates/*.html
var templateFS embed.FS

type pageSet struct {
	*template.Template
}

var pages = map[string]*pageSet{
	"login":      parsePage("login.html"),
	"users":      parsePage("users.html"),
	"usercreate": parsePage("user_create.html"),
	"user_login": parsePage("user_login.html"),
}

func parsePage(file string) *pageSet {
	return &pageSet{
		template.Must(template.Must(template.New("").ParseFS(templateFS, "templates/base.html")).ParseFS(templateFS, "templates/"+file)),
	}
}

type userWithToken struct {
	ID          int
	Name        string
	CreatedAt   string
	TokenPrefix string
}

type usersPageData struct {
	Users    []userWithToken
	NewToken string
	Error    string
}

type AdminHandler struct {
	userSvc *service.UserService
	db      *sqlite.DB
}

func NewAdminHandler(db *sqlite.DB, userSvc *service.UserService) *AdminHandler {
	return &AdminHandler{userSvc: userSvc, db: db}
}

func (h *AdminHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/admin/login", h.handleLogin)
	mux.HandleFunc("/admin/users", h.authWrap(h.handleUsers))
	mux.HandleFunc("/admin/users/create", h.authWrap(h.handleUserCreate))
	mux.HandleFunc("/admin/users/refresh", h.authWrap(h.handleTokenRefresh))
	mux.HandleFunc("/admin/users/delete", h.authWrap(h.handleUserDelete))
	mux.HandleFunc("/admin/logout", h.handleLogout)
	mux.HandleFunc("/admin/", h.handleIndex)
}

func (h *AdminHandler) authWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkSession(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		fn(w, r)
	}
}

func (h *AdminHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/" && r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}
	if checkSession(r) {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

func (h *AdminHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		pages["login"].ExecuteTemplate(w, "base", nil)
		return
	}
	password := r.FormValue("password")
	storedHash, _ := h.db.GetSetting("admin_password_hash")
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		pages["login"].ExecuteTemplate(w, "base", map[string]string{"Error": "密码错误"})
		return
	}
	setSession(w)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) buildUsers() []userWithToken {
	users, _ := h.userSvc.ListUsers()
	var data []userWithToken
	for _, u := range users {
		data = append(data, userWithToken{
			ID:          u.ID,
			Name:        u.Name,
			CreatedAt:   u.CreatedAt,
			TokenPrefix: h.userSvc.GetActiveTokenPrefix(u.ID),
		})
	}
	return data
}

func (h *AdminHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	pages["users"].ExecuteTemplate(w, "base", usersPageData{Users: h.buildUsers()})
}

func (h *AdminHandler) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		pages["usercreate"].ExecuteTemplate(w, "base", nil)
		return
	}
	name := r.FormValue("name")
	u, token, err := h.userSvc.CreateUser(name)
	if err != nil {
		pages["usercreate"].ExecuteTemplate(w, "base", map[string]string{"Error": err.Error()})
		return
	}
	pages["usercreate"].ExecuteTemplate(w, "base", map[string]any{
		"Success": true,
		"Name":    u.Name,
		"Token":   token,
	})
}

func (h *AdminHandler) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	userID, _ := strconv.Atoi(r.FormValue("user_id"))
	if userID == 0 {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}
	token, err := h.userSvc.RefreshToken(userID)
	if err != nil {
		pages["users"].ExecuteTemplate(w, "base", usersPageData{Users: h.buildUsers(), Error: err.Error()})
		return
	}
	pages["users"].ExecuteTemplate(w, "base", usersPageData{Users: h.buildUsers(), NewToken: token})
}

func (h *AdminHandler) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	userID, _ := strconv.Atoi(r.FormValue("user_id"))
	if err := h.userSvc.DeleteUser(userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
