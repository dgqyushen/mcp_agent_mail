package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcp "github.com/mark3labs/mcp-go/mcp"

	"agent-mail/model"
	"agent-mail/service"
)

type Handler struct {
	mailbox    *service.MailboxService
	email      *service.EmailService
	send       *service.SendService
	autoReply  *service.AutoReplyService
	webhook    *service.WebhookService
	attachment *service.AttachmentService
}

func NewHandler(
	mailbox *service.MailboxService,
	email *service.EmailService,
	send *service.SendService,
	autoReply *service.AutoReplyService,
	webhook *service.WebhookService,
	attachment *service.AttachmentService,
) *Handler {
	return &Handler{
		mailbox:    mailbox,
		email:      email,
		send:       send,
		autoReply:  autoReply,
		webhook:    webhook,
		attachment: attachment,
	}
}

func (h *Handler) HandleToolCall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID := GetUserID(ctx)
	switch req.Params.Name {
	case ToolListMailboxes:
		return h.listMailboxes(userID)
	case ToolAddMailbox:
		return h.addMailbox(userID, req)
	case ToolRemoveMailbox:
		return h.removeMailbox(userID, req)
	case ToolSwitchMailbox:
		return h.switchMailbox(userID, req)
	case ToolValidateMailbox:
		return h.validateMailbox(userID, req)
	case ToolListEmails:
		return h.listEmails(userID, req)
	case ToolGetEmail:
		return h.getEmail(userID, req)
	case ToolSearchEmails:
		return h.searchEmails(userID, req)
	case ToolDeleteEmail:
		return h.deleteEmail(userID, req)
	case ToolClearInbox:
		return h.clearInbox(userID, req)
	case ToolSendMail:
		return h.sendMail(userID, req)
	case ToolCheckBalance:
		return h.checkBalance(userID, req)
	case ToolListSent:
		return h.listSent(userID, req)
	case ToolDeleteSent:
		return h.deleteSent(userID, req)
	case ToolClearSent:
		return h.clearSent(userID, req)
	case ToolGetAutoReply:
		return h.getAutoReply(userID, req)
	case ToolSetAutoReply:
		return h.setAutoReply(userID, req)
	case ToolGetWebhook:
		return h.getWebhook(userID, req)
	case ToolSetWebhook:
		return h.setWebhook(userID, req)
	case ToolListAttach:
		return h.listAttachments(userID, req)
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Params.Name)
	}
}

func getArgs(req mcp.CallToolRequest) map[string]any {
	args, _ := req.Params.Arguments.(map[string]any)
	return args
}

func strArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intArg(args map[string]any, key string) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func boolArg(args map[string]any, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func requireArgs(args map[string]any, keys ...string) error {
	for _, k := range keys {
		v, ok := args[k]
		if !ok || v == nil {
			return fmt.Errorf("missing required parameter: %q", k)
		}
		switch val := v.(type) {
		case string:
			if val == "" {
				return fmt.Errorf("parameter %q must not be empty", k)
			}
		case float64:
			if val == 0 {
				return fmt.Errorf("parameter %q must not be zero", k)
			}
		}
	}
	return nil
}

func toResult(v any) *mcp.CallToolResult {
	data, err := json.Marshal(v)
	if err != nil {
		return errorResult(fmt.Errorf("failed to marshal result: %w", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: string(data)},
		},
	}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: err.Error()},
		},
	}
}

// --- handlers ---

func (h *Handler) listMailboxes(userID int) (*mcp.CallToolResult, error) {
	list, err := h.mailbox.List(userID)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(list), nil
}

func (h *Handler) addMailbox(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "alias", "name", "base_url", "auth_data"); err != nil {
		return errorResult(err), nil
	}
	err := h.mailbox.Add(
		userID,
		strArg(args, "alias"),
		strArg(args, "name"),
		strArg(args, "provider_type"),
		strArg(args, "base_url"),
		strArg(args, "auth_data"),
	)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "ok"}), nil
}

func (h *Handler) removeMailbox(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.mailbox.Remove(userID, alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "ok"}), nil
}

func (h *Handler) switchMailbox(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.mailbox.Switch(userID, alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "switched", "alias": alias}), nil
}

func (h *Handler) validateMailbox(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	settings, err := h.mailbox.Validate(userID, alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(settings), nil
}

func (h *Handler) listEmails(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.email.List(userID, strArg(args, "alias"), intArg(args, "limit"), intArg(args, "offset"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) getEmail(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.email.Get(userID, strArg(args, "alias"), intArg(args, "id"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) searchEmails(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.email.Search(userID, strArg(args, "alias"), strArg(args, "query"), intArg(args, "limit"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) deleteEmail(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "id"); err != nil {
		return errorResult(err), nil
	}
	if err := h.email.Delete(userID, strArg(args, "alias"), intArg(args, "id")); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]any{"status": "deleted", "id": intArg(args, "id")}), nil
}

func (h *Handler) clearInbox(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.email.Clear(userID, alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "inbox cleared"}), nil
}

func (h *Handler) sendMail(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "to_mail", "subject", "content"); err != nil {
		return errorResult(err), nil
	}
	body := &model.SendMailBody{
		FromName: strArg(args, "from_name"),
		ToMail:   strArg(args, "to_mail"),
		ToName:   strArg(args, "to_name"),
		Subject:  strArg(args, "subject"),
		Content:  strArg(args, "content"),
		IsHTML:   boolArg(args, "is_html"),
	}
	if err := h.send.Send(userID, strArg(args, "alias"), body); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "sent"}), nil
}

func (h *Handler) checkBalance(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	balance, err := h.send.CheckBalance(userID, alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]int{"balance": balance}), nil
}

func (h *Handler) listSent(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.send.ListSent(userID, strArg(args, "alias"), intArg(args, "limit"), intArg(args, "offset"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) deleteSent(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "id"); err != nil {
		return errorResult(err), nil
	}
	if err := h.send.DeleteSent(userID, strArg(args, "alias"), intArg(args, "id")); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]any{"status": "deleted", "id": intArg(args, "id")}), nil
}

func (h *Handler) clearSent(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.send.ClearSent(userID, alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "sent items cleared"}), nil
}

func (h *Handler) getAutoReply(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	cfg, err := h.autoReply.Get(userID, alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(cfg), nil
}

func (h *Handler) setAutoReply(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "enabled"); err != nil {
		return errorResult(err), nil
	}
	cfg := &model.AutoReplyConfig{
		Name:         strArg(args, "name"),
		Subject:      strArg(args, "subject"),
		SourcePrefix: strArg(args, "source_prefix"),
		Message:      strArg(args, "message"),
		Enabled:      boolArg(args, "enabled"),
	}
	if err := h.autoReply.Set(userID, strArg(args, "alias"), cfg); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "ok"}), nil
}

func (h *Handler) getWebhook(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	cfg, err := h.webhook.Get(userID, alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(cfg), nil
}

func (h *Handler) setWebhook(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "url", "events"); err != nil {
		return errorResult(err), nil
	}
	var events []string
	if raw, ok := args["events"]; ok {
		if arr, ok := raw.([]any); ok {
			for _, e := range arr {
				if s, ok := e.(string); ok {
					events = append(events, s)
				}
			}
		}
	}
	cfg := &model.WebhookSettings{
		URL:    strArg(args, "url"),
		Events: events,
	}
	if err := h.webhook.Set(userID, strArg(args, "alias"), cfg); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "ok"}), nil
}

func (h *Handler) listAttachments(userID int, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	result, err := h.attachment.List(userID, alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}
