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
	switch req.Params.Name {
	case ToolListMailboxes:
		return h.listMailboxes()
	case ToolAddMailbox:
		return h.addMailbox(req)
	case ToolRemoveMailbox:
		return h.removeMailbox(req)
	case ToolSwitchMailbox:
		return h.switchMailbox(req)
	case ToolValidateMailbox:
		return h.validateMailbox(req)
	case ToolListEmails:
		return h.listEmails(req)
	case ToolGetEmail:
		return h.getEmail(req)
	case ToolSearchEmails:
		return h.searchEmails(req)
	case ToolDeleteEmail:
		return h.deleteEmail(req)
	case ToolClearInbox:
		return h.clearInbox(req)
	case ToolSendMail:
		return h.sendMail(req)
	case ToolCheckBalance:
		return h.checkBalance(req)
	case ToolListSent:
		return h.listSent(req)
	case ToolDeleteSent:
		return h.deleteSent(req)
	case ToolClearSent:
		return h.clearSent(req)
	case ToolGetAutoReply:
		return h.getAutoReply(req)
	case ToolSetAutoReply:
		return h.setAutoReply(req)
	case ToolGetWebhook:
		return h.getWebhook(req)
	case ToolSetWebhook:
		return h.setWebhook(req)
	case ToolListAttach:
		return h.listAttachments(req)
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

func (h *Handler) listMailboxes() (*mcp.CallToolResult, error) {
	list, err := h.mailbox.List()
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(list), nil
}

func (h *Handler) addMailbox(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	err := h.mailbox.Add(
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

func (h *Handler) removeMailbox(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.mailbox.Remove(alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "ok"}), nil
}

func (h *Handler) switchMailbox(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.mailbox.Switch(alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "switched", "alias": alias}), nil
}

func (h *Handler) validateMailbox(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	settings, err := h.mailbox.Validate(alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(settings), nil
}

func (h *Handler) listEmails(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.email.List(strArg(args, "alias"), intArg(args, "limit"), intArg(args, "offset"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) getEmail(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.email.Get(strArg(args, "alias"), intArg(args, "id"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) searchEmails(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.email.Search(strArg(args, "alias"), strArg(args, "query"), intArg(args, "limit"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) deleteEmail(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "id"); err != nil {
		return errorResult(err), nil
	}
	if err := h.email.Delete(strArg(args, "alias"), intArg(args, "id")); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]any{"status": "deleted", "id": intArg(args, "id")}), nil
}

func (h *Handler) clearInbox(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.email.Clear(alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "inbox cleared"}), nil
}

func (h *Handler) sendMail(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if err := h.send.Send(strArg(args, "alias"), body); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "sent"}), nil
}

func (h *Handler) checkBalance(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	balance, err := h.send.CheckBalance(alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]int{"balance": balance}), nil
}

func (h *Handler) listSent(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	result, err := h.send.ListSent(strArg(args, "alias"), intArg(args, "limit"), intArg(args, "offset"))
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}

func (h *Handler) deleteSent(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	if err := requireArgs(args, "id"); err != nil {
		return errorResult(err), nil
	}
	if err := h.send.DeleteSent(strArg(args, "alias"), intArg(args, "id")); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]any{"status": "deleted", "id": intArg(args, "id")}), nil
}

func (h *Handler) clearSent(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	if err := h.send.ClearSent(alias); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "sent items cleared"}), nil
}

func (h *Handler) getAutoReply(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	cfg, err := h.autoReply.Get(alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(cfg), nil
}

func (h *Handler) setAutoReply(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if err := h.autoReply.Set(strArg(args, "alias"), cfg); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "ok"}), nil
}

func (h *Handler) getWebhook(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	cfg, err := h.webhook.Get(alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(cfg), nil
}

func (h *Handler) setWebhook(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if err := h.webhook.Set(strArg(args, "alias"), cfg); err != nil {
		return errorResult(err), nil
	}
	return toResult(map[string]string{"status": "ok"}), nil
}

func (h *Handler) listAttachments(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alias := strArg(getArgs(req), "alias")
	result, err := h.attachment.List(alias)
	if err != nil {
		return errorResult(err), nil
	}
	return toResult(result), nil
}
