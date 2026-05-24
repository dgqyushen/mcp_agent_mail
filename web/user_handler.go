package web

import (
	"net/http"

	"agent-mail/model"
	"agent-mail/service"
)

type mailboxWithDefault struct {
	model.MailboxInfo
	IsDefault bool
}

type mailboxesPageData struct {
	Mailboxes []mailboxWithDefault
	Error     string
	Success   string
}

type UserHandler struct {
	userSvc    *service.UserService
	mailboxSvc *service.MailboxService
}

func NewUserHandler(userSvc *service.UserService, mailboxSvc *service.MailboxService) *UserHandler {
	return &UserHandler{userSvc: userSvc, mailboxSvc: mailboxSvc}
}

func (h *UserHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/user/login", h.handleLogin)
	mux.HandleFunc("/user/logout", h.handleLogout)
	mux.HandleFunc("/user/mailboxes", h.authWrap(h.handleMailboxes))
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

func (h *UserHandler) handleMailboxes(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	infos, err := h.mailboxSvc.List(userID)
	if err != nil {
		pages["user_mailboxes"].ExecuteTemplate(w, "base", mailboxesPageData{Error: err.Error()})
		return
	}
	def := h.mailboxSvc.Default(userID)
	var mails []mailboxWithDefault
	for _, info := range infos {
		mails = append(mails, mailboxWithDefault{
			MailboxInfo: info,
			IsDefault:   info.Alias == def,
		})
	}
	pages["user_mailboxes"].ExecuteTemplate(w, "base", mailboxesPageData{Mailboxes: mails})
}

// getUserID extracts the user ID from a valid user session.
// Must only be called inside authWrap'd handlers.
func getUserID(r *http.Request) int {
	return 0 // placeholder — will be replaced in Task 8 where user ID is stored in session
}
