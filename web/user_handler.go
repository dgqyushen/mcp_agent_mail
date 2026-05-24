package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"agent-mail/model"
	"agent-mail/provider"
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

type mailboxFormPageData struct {
	Providers    []provider.ProviderFormInfo
	Fields       []provider.FieldDef
	FieldValues  func(key string) string
	ProviderType string
	Alias        string
	Name         string
	IsEdit       bool
	Error        string
	Success      string
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
	mux.HandleFunc("/user/mailboxes/add", h.authWrap(h.handleMailboxAdd))
	mux.HandleFunc("/user/mailboxes/edit", h.authWrap(h.handleMailboxEdit))
	mux.HandleFunc("/user/mailboxes/delete", h.authWrap(h.handleMailboxDelete))
	mux.HandleFunc("/user/mailboxes/switch", h.authWrap(h.handleMailboxSwitch))
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
	userID, err := h.userSvc.ValidateToken(token)
	if err != nil {
		pages["user_login"].ExecuteTemplate(w, "base", map[string]string{"Error": "Token 无效或已过期"})
		return
	}
	setUserSession(w, userID)
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

func (h *UserHandler) buildFormFields(providerType string, authDataJSON string) []provider.FieldDef {
	info := provider.GetProviderFormInfo(providerType)
	if info == nil {
		return nil
	}
	return info.Fields
}

func (h *UserHandler) parseFormFieldsToJSON(r *http.Request, providerType string) (baseURL string, authData string, err error) {
	info := provider.GetProviderFormInfo(providerType)
	if info == nil {
		return "", "", fmt.Errorf("unknown provider type: %s", providerType)
	}
	authMap := make(map[string]string)
	for _, f := range info.Fields {
		val := r.FormValue(f.Key)
		if f.Section == "base_url" {
			baseURL = val
		} else {
			authMap[f.Key] = val
		}
	}
	b, err := json.Marshal(authMap)
	if err != nil {
		return "", "", err
	}
	return baseURL, string(b), nil
}

func (h *UserHandler) handleMailboxAdd(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	if r.Method == "GET" {
		providerType := r.FormValue("provider_type")
		infos := provider.GetProviderFormInfos()
		data := mailboxFormPageData{
			Providers:    infos,
			ProviderType: providerType,
			IsEdit:       false,
			FieldValues:  func(key string) string { return "" },
		}
		if providerType != "" {
			data.Fields = h.buildFormFields(providerType, "")
		}
		pages["user_mailbox_form"].ExecuteTemplate(w, "base", data)
		return
	}

	providerType := r.FormValue("provider_type")
	if providerType == "" {
		providerType = r.FormValue("provider_type_hidden")
	}
	alias := r.FormValue("alias")
	name := r.FormValue("name")
	baseURL, authData, err := h.parseFormFieldsToJSON(r, providerType)
	if err != nil {
		pages["user_mailbox_form"].ExecuteTemplate(w, "base", mailboxFormPageData{Error: err.Error()})
		return
	}

	if err := h.mailboxSvc.Add(userID, alias, name, providerType, baseURL, authData); err != nil {
		pages["user_mailbox_form"].ExecuteTemplate(w, "base", mailboxFormPageData{
			Error: err.Error(),
			Alias: alias, Name: name, ProviderType: providerType,
		})
		return
	}
	http.Redirect(w, r, "/user/mailboxes", http.StatusSeeOther)
}

func (h *UserHandler) handleMailboxEdit(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	alias := r.FormValue("alias")
	if alias == "" {
		http.Error(w, "missing alias", http.StatusBadRequest)
		return
	}

	rec, err := h.mailboxSvc.Resolve(userID, alias)
	if err != nil {
		pages["user_mailbox_form"].ExecuteTemplate(w, "base", mailboxFormPageData{Error: err.Error()})
		return
	}

	var existingAuth map[string]string
	json.Unmarshal([]byte(rec.AuthData), &existingAuth)
	if existingAuth == nil {
		existingAuth = make(map[string]string)
	}

	fieldValues := func(key string) string {
		info := provider.GetProviderFormInfo(rec.ProviderType)
		if info != nil {
			for _, f := range info.Fields {
				if f.Key == key && f.Section == "base_url" {
					return rec.BaseURL
				}
			}
		}
		return existingAuth[key]
	}

	if r.Method == "GET" {
		infos := provider.GetProviderFormInfos()
		data := mailboxFormPageData{
			Providers:    infos,
			Fields:       h.buildFormFields(rec.ProviderType, rec.AuthData),
			ProviderType: rec.ProviderType,
			Alias:        rec.Alias,
			Name:         rec.Name,
			IsEdit:       true,
			FieldValues:  fieldValues,
		}
		pages["user_mailbox_form"].ExecuteTemplate(w, "base", data)
		return
	}

	providerType := r.FormValue("provider_type")
	if providerType == "" {
		providerType = rec.ProviderType
	}
	name := r.FormValue("name")
	baseURL, authData, err := h.parseFormFieldsToJSON(r, providerType)
	if err != nil {
		pages["user_mailbox_form"].ExecuteTemplate(w, "base", mailboxFormPageData{Error: err.Error()})
		return
	}

	if err := h.mailboxSvc.Update(userID, alias, name, providerType, baseURL, authData); err != nil {
		pages["user_mailbox_form"].ExecuteTemplate(w, "base", mailboxFormPageData{Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/user/mailboxes", http.StatusSeeOther)
}

func (h *UserHandler) handleMailboxDelete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	alias := r.FormValue("alias")
	if err := h.mailboxSvc.Remove(userID, alias); err != nil {
		if r.Method == "POST" {
			http.Redirect(w, r, "/user/mailboxes?error="+err.Error(), http.StatusSeeOther)
			return
		}
		pages["user_mailboxes"].ExecuteTemplate(w, "base", mailboxesPageData{Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/user/mailboxes", http.StatusSeeOther)
}

func (h *UserHandler) handleMailboxSwitch(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	alias := r.FormValue("alias")
	if err := h.mailboxSvc.Switch(userID, alias); err != nil {
		pages["user_mailboxes"].ExecuteTemplate(w, "base", mailboxesPageData{Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/user/mailboxes", http.StatusSeeOther)
}
