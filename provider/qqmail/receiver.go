package qqmail

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"agent-mail/model"
	"agent-mail/provider"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

type QQmailReceiver struct {
	auth QQmailAuthData
}

func (r *QQmailReceiver) Validate() error {
	if r.auth.Username == "" || r.auth.Password == "" {
		return fmt.Errorf("qqmail: auth credentials required")
	}
	client, err := newIMAPClient(r.auth)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Logout().Wait()
}

func (r *QQmailReceiver) GetSettings() (*model.SettingsResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &model.SettingsResponse{
		Address:     r.auth.Username,
		SendBalance: 0,
	}, nil
}

func (r *QQmailReceiver) ListEmails(limit, offset int) (*model.PaginatedResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	client, err := newIMAPClient(r.auth)
	if err != nil {
		return nil, fmt.Errorf("qqmail: %w", err)
	}
	defer client.Close()

	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("qqmail select inbox: %w", err)
	}

	searchData, err := client.UIDSearch(&imap.SearchCriteria{}, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("qqmail search: %w", err)
	}
	uids := searchData.AllUIDs()

	sort.Slice(uids, func(i, j int) bool { return uids[i] > uids[j] })

	total := len(uids)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	if start >= end {
		return &model.PaginatedResult{Results: []model.ParsedMail{}, Count: total}, nil
	}

	pageUIDs := uids[start:end]
	messages, err := client.Fetch(imap.UIDSetNum(pageUIDs...), &imap.FetchOptions{
		Envelope: true,
		UID:      true,
	}).Collect()
	if err != nil {
		return nil, fmt.Errorf("qqmail fetch: %w", err)
	}

	results := make([]model.ParsedMail, 0, len(messages))
	for _, msg := range messages {
		results = append(results, bufferToParsedMail(msg))
	}

	return &model.PaginatedResult{Results: results, Count: total}, nil
}

func (r *QQmailReceiver) GetEmail(id int) (*model.ParsedMail, error) {
	client, err := newIMAPClient(r.auth)
	if err != nil {
		return nil, fmt.Errorf("qqmail: %w", err)
	}
	defer client.Close()

	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("qqmail select inbox: %w", err)
	}

	bodySection := &imap.FetchItemBodySection{}
	messages, err := client.Fetch(imap.UIDSetNum(imap.UID(id)), &imap.FetchOptions{
		Envelope:    true,
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}).Collect()
	if err != nil {
		return nil, fmt.Errorf("qqmail fetch: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("qqmail: message %d not found", id)
	}

	parsed := bufferToParsedMail(messages[0])
	if raw := messages[0].FindBodySection(bodySection); raw != nil {
		text, html, atts, parseErr := parseMIMEMessage(raw)
		if parseErr == nil {
			parsed.Text = text
			parsed.HTML = html
			parsed.Attachments = atts
		} else {
			parsed.Text = string(raw)
		}
	}

	return &parsed, nil
}

func (r *QQmailReceiver) DeleteEmail(id int) error {
	client, err := newIMAPClient(r.auth)
	if err != nil {
		return fmt.Errorf("qqmail: %w", err)
	}
	defer client.Close()

	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		return fmt.Errorf("qqmail select inbox: %w", err)
	}

	uidSet := imap.UIDSetNum(imap.UID(id))
	storeFlags := &imap.StoreFlags{
		Op:    imap.StoreFlagsSet,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if err := client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("qqmail store: %w", err)
	}
	if err := client.Expunge().Close(); err != nil {
		return fmt.Errorf("qqmail expunge: %w", err)
	}
	return nil
}

func (r *QQmailReceiver) ClearInbox() error {
	client, err := newIMAPClient(r.auth)
	if err != nil {
		return fmt.Errorf("qqmail: %w", err)
	}
	defer client.Close()

	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		return fmt.Errorf("qqmail select inbox: %w", err)
	}

	searchData, err := client.UIDSearch(&imap.SearchCriteria{}, nil).Wait()
	if err != nil {
		return fmt.Errorf("qqmail search: %w", err)
	}
	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return nil
	}

	uidSet := imap.UIDSetNum(uids...)
	storeFlags := &imap.StoreFlags{
		Op:    imap.StoreFlagsSet,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if err := client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("qqmail store: %w", err)
	}
	if err := client.Expunge().Close(); err != nil {
		return fmt.Errorf("qqmail expunge: %w", err)
	}
	return nil
}

func (r *QQmailReceiver) GetAutoReply() (*model.AutoReplyConfig, error) {
	return nil, provider.ErrCapNotSupported
}

func (r *QQmailReceiver) SetAutoReply(cfg *model.AutoReplyConfig) error {
	return provider.ErrCapNotSupported
}

func (r *QQmailReceiver) GetWebhook() (*model.WebhookSettings, error) {
	return nil, provider.ErrCapNotSupported
}

func (r *QQmailReceiver) SetWebhook(cfg *model.WebhookSettings) error {
	return provider.ErrCapNotSupported
}

func (r *QQmailReceiver) ListAttachments() (*model.AttachmentListResult, error) {
	return nil, provider.ErrCapNotSupported
}

func bufferToParsedMail(msg *imapclient.FetchMessageBuffer) model.ParsedMail {
	sender := ""
	to := ""
	subject := ""
	dateStr := ""

	if msg.Envelope != nil {
		if len(msg.Envelope.From) > 0 {
			addr := msg.Envelope.From[0]
			if addr.Name != "" {
				sender = fmt.Sprintf("%s <%s@%s>", addr.Name, addr.Mailbox, addr.Host)
			} else {
				sender = fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
			}
		}
		if len(msg.Envelope.To) > 0 {
			addr := msg.Envelope.To[0]
			if addr.Name != "" {
				to = fmt.Sprintf("%s <%s@%s>", addr.Name, addr.Mailbox, addr.Host)
			} else {
				to = fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
			}
		}
		subject = msg.Envelope.Subject
		if !msg.Envelope.Date.IsZero() {
			dateStr = msg.Envelope.Date.Format(time.RFC3339)
		}
	}

	return model.ParsedMail{
		ID:        int(msg.UID),
		MessageID: fmt.Sprintf("%d", msg.UID),
		Source:    sender,
		Sender:    sender,
		To:        to,
		Subject:   subject,
		CreatedAt: dateStr,
	}
}

type QQmailSender struct {
	auth QQmailAuthData
}

func (s *QQmailSender) Validate() error {
	if s.auth.Username == "" || s.auth.Password == "" {
		return fmt.Errorf("qqmail sender: auth credentials required")
	}
	return nil
}

func (s *QQmailSender) SendMail(body *model.SendMailBody) error {
	return provider.ErrCapNotSupported
}

func (s *QQmailSender) CheckSendBalance() (int, error) {
	return 0, provider.ErrCapNotSupported
}

func (s *QQmailSender) ListSent(limit, offset int) (*model.SendboxResult, error) {
	return nil, provider.ErrCapNotSupported
}

func (s *QQmailSender) DeleteSent(id int) error {
	return provider.ErrCapNotSupported
}

func (s *QQmailSender) ClearSent() error {
	return provider.ErrCapNotSupported
}

func parseMIMEMessage(raw []byte) (text, html string, atts []model.Attachment, err error) {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return "", "", nil, err
	}

	for {
		p, partErr := mr.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			break
		}

		body, _ := io.ReadAll(p.Body)

		switch h := p.Header.(type) {
		case *mail.AttachmentHeader:
			contentType, _, _ := h.ContentType()
			cd, _, _ := h.ContentDisposition()
			filename, _ := h.Filename()
			atts = append(atts, model.Attachment{
				Filename:    filename,
				MimeType:    contentType,
				Disposition: cd,
				Size:        len(body),
			})
		case *mail.InlineHeader:
			contentType, _, _ := h.ContentType()
			if strings.HasPrefix(contentType, "text/plain") && text == "" {
				text = string(body)
			} else if strings.HasPrefix(contentType, "text/html") && html == "" {
				html = string(body)
			}
		}
	}

	return text, html, atts, nil
}
