package gmail

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"agent-mail/model"
	"agent-mail/provider"

	"google.golang.org/api/gmail/v1"
)

type GmailSender struct {
	srv  *gmail.Service
	auth GmailAuthData
}

func (s *GmailSender) Validate() error {
	if s.srv == nil {
		return fmt.Errorf("gmail service not initialized")
	}
	return nil
}

func (s *GmailSender) SendMail(body *model.SendMailBody) error {
	return provider.ErrCapNotSupported
}

func (s *GmailSender) CheckSendBalance() (int, error) {
	return 0, nil
}

func (s *GmailSender) ListSent(limit, offset int) (*model.SendboxResult, error) {
	return &model.SendboxResult{}, nil
}

func (s *GmailSender) DeleteSent(id int) error {
	return provider.ErrCapNotSupported
}

func (s *GmailSender) ClearSent() error {
	return provider.ErrCapNotSupported
}

type GmailReceiver struct {
	srv  *gmail.Service
	auth GmailAuthData
}

func (r *GmailReceiver) GetSettings() (*model.SettingsResponse, error) {
	profile, err := r.srv.Users.GetProfile("me").Do()
	if err != nil {
		return nil, fmt.Errorf("get gmail profile: %w", err)
	}
	return &model.SettingsResponse{
		Address:     profile.EmailAddress,
		SendBalance: 0,
	}, nil
}

func (r *GmailReceiver) ListEmails(limit, offset int) (*model.PaginatedResult, error) {
	var allIDs []string
	var total int
	pageToken := ""
	firstPage := true

	for len(allIDs) < offset+limit {
		call := r.srv.Users.Messages.List("me").MaxResults(500)
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list gmail messages: %w", err)
		}
		if firstPage {
			total = int(resp.ResultSizeEstimate)
			firstPage = false
		}
		for _, m := range resp.Messages {
			allIDs = append(allIDs, m.Id)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	start := offset
	if start > len(allIDs) {
		start = len(allIDs)
	}
	end := start + limit
	if end > len(allIDs) {
		end = len(allIDs)
	}
	window := allIDs[start:end]

	var results []model.ParsedMail
	for _, id := range window {
		parsed, err := r.getParsedMail(id)
		if err != nil {
			return nil, err
		}
		results = append(results, *parsed)
	}

	return &model.PaginatedResult{
		Results: results,
		Count:   total,
	}, nil
}

func (r *GmailReceiver) GetEmail(id int) (*model.ParsedMail, error) {
	msgID := fmt.Sprintf("%d", id)
	msg, err := r.srv.Users.Messages.Get("me", msgID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("get gmail message: %w", err)
	}
	parsed := parseMessage(msg)
	parsed.ID = id
	return parsed, nil
}

func (r *GmailReceiver) DeleteEmail(id int) error {
	msgID := fmt.Sprintf("%d", id)
	_, err := r.srv.Users.Messages.Trash("me", msgID).Do()
	if err != nil {
		return fmt.Errorf("trash gmail message: %w", err)
	}
	return nil
}

func (r *GmailReceiver) ClearInbox() error {
	var allIDs []string
	pageToken := ""
	for {
		call := r.srv.Users.Messages.List("me").MaxResults(500)
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return fmt.Errorf("list gmail messages: %w", err)
		}
		for _, m := range resp.Messages {
			allIDs = append(allIDs, m.Id)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	if len(allIDs) == 0 {
		return nil
	}

	err := r.srv.Users.Messages.BatchDelete("me", &gmail.BatchDeleteMessagesRequest{
		Ids: allIDs,
	}).Do()
	if err != nil {
		return fmt.Errorf("batch delete gmail messages: %w", err)
	}
	return nil
}

func (r *GmailReceiver) GetAutoReply() (*model.AutoReplyConfig, error) {
	vacation, err := r.srv.Users.Settings.GetVacation("me").Do()
	if err != nil {
		return nil, fmt.Errorf("get gmail vacation: %w", err)
	}
	return &model.AutoReplyConfig{
		Enabled: vacation.EnableAutoReply,
		Subject: vacation.ResponseSubject,
		Message: vacation.ResponseBodyPlainText,
	}, nil
}

func (r *GmailReceiver) SetAutoReply(cfg *model.AutoReplyConfig) error {
	vacation := &gmail.VacationSettings{
		EnableAutoReply:     cfg.Enabled,
		ResponseSubject:     cfg.Subject,
		ResponseBodyPlainText: cfg.Message,
	}
	_, err := r.srv.Users.Settings.UpdateVacation("me", vacation).Do()
	if err != nil {
		return fmt.Errorf("set gmail vacation: %w", err)
	}
	return nil
}

func (r *GmailReceiver) GetWebhook() (*model.WebhookSettings, error) {
	return nil, provider.ErrCapNotSupported
}

func (r *GmailReceiver) SetWebhook(cfg *model.WebhookSettings) error {
	return provider.ErrCapNotSupported
}

func (r *GmailReceiver) ListAttachments() (*model.AttachmentListResult, error) {
	return nil, provider.ErrCapNotSupported
}

func (r *GmailReceiver) Validate() error {
	if r.srv == nil {
		return fmt.Errorf("gmail service not initialized")
	}
	_, err := r.srv.Users.GetProfile("me").Do()
	if err != nil {
		return fmt.Errorf("gmail validate: %w", err)
	}
	return nil
}

func (r *GmailReceiver) getParsedMail(id string) (*model.ParsedMail, error) {
	msg, err := r.srv.Users.Messages.Get("me", id).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("get gmail message: %w", err)
	}
	return parseMessage(msg), nil
}

func decodeBase64(s string) string {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func extractBody(parts []*gmail.MessagePart) (text, html string) {
	for _, part := range parts {
		if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
			text = decodeBase64(part.Body.Data)
		} else if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
			html = decodeBase64(part.Body.Data)
		}
		if len(part.Parts) > 0 {
			t, h := extractBody(part.Parts)
			if text == "" {
				text = t
			}
			if html == "" {
				html = h
			}
		}
	}
	return
}

func extractAttachments(parts []*gmail.MessagePart) []model.Attachment {
	var atts []model.Attachment
	for _, part := range parts {
		if part.Filename != "" && part.Body != nil && part.Body.Size > 0 {
			disposition := ""
			for _, h := range part.Headers {
				if h.Name == "Content-Disposition" {
					disposition = h.Value
					break
				}
			}
			atts = append(atts, model.Attachment{
				Filename:    part.Filename,
				MimeType:    part.MimeType,
				Disposition: disposition,
				Size:        int(part.Body.Size),
			})
		}
		if len(part.Parts) > 0 {
			atts = append(atts, extractAttachments(part.Parts)...)
		}
	}
	return atts
}

func getHeader(headers []*gmail.MessagePartHeader, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

func parseMessage(msg *gmail.Message) *model.ParsedMail {
	headers := msg.Payload.Headers
	from := getHeader(headers, "From")
	to := getHeader(headers, "To")
	subject := getHeader(headers, "Subject")

	text, html := "", ""
	atts := extractAttachments(msg.Payload.Parts)

	if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		if msg.Payload.MimeType == "text/plain" {
			text = decodeBase64(msg.Payload.Body.Data)
		} else if msg.Payload.MimeType == "text/html" {
			html = decodeBase64(msg.Payload.Body.Data)
		}
	}
	t, h := extractBody(msg.Payload.Parts)
	if text == "" {
		text = t
	}
	if html == "" {
		html = h
	}

	createdAt := ""
	if msg.InternalDate > 0 {
		createdAt = time.UnixMilli(msg.InternalDate).Format(time.RFC3339)
	}

	id, _ := strconv.ParseInt(msg.Id, 10, 64)

	return &model.ParsedMail{
		ID:          int(id),
		MessageID:   msg.Id,
		Source:      from,
		To:          to,
		CreatedAt:   createdAt,
		Sender:      from,
		Subject:     subject,
		Text:        text,
		HTML:        html,
		Attachments: atts,
	}
}
