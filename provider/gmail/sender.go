package gmail

import (
	"fmt"
	"net/smtp"
	"strconv"
	"time"

	"agent-mail/model"
	"agent-mail/provider"

	"google.golang.org/api/gmail/v1"
)

type GmailSender struct {
	srv       *gmail.Service
	auth      GmailAuthData
	fromEmail string
}

func (s *GmailSender) Validate() error {
	if s.auth.AccessToken == "" {
		return fmt.Errorf("gmail sender: access token not set")
	}
	return nil
}

func (s *GmailSender) SendMail(body *model.SendMailBody) error {
	if s.fromEmail == "" {
		profile, err := s.srv.Users.GetProfile("me").Do()
		if err != nil {
			return fmt.Errorf("gmail get profile for from: %w", err)
		}
		s.fromEmail = profile.EmailAddress
	}

	msg := provider.BuildSMTPMessage(body.FromName, s.fromEmail, body.ToMail, body.ToName, body.Subject, body.Content, body.IsHTML)
	return s.sendWithXOAUTH2(body.ToMail, msg)
}

func (s *GmailSender) CheckSendBalance() (int, error) {
	return 0, provider.ErrCapNotSupported
}

func (s *GmailSender) ListSent(limit, offset int) (*model.SendboxResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var allIDs []string
	var total int
	pageToken := ""
	firstPage := true

	for len(allIDs) < offset+limit {
		call := s.srv.Users.Messages.List("me").MaxResults(500).Q("in:sent")
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list sent messages: %w", err)
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

	var results []model.SendboxItem
	for _, id := range window {
		msg, err := s.srv.Users.Messages.Get("me", id).Format("full").Do()
		if err != nil {
			return nil, fmt.Errorf("get sent message %s: %w", id, err)
		}
		results = append(results, *messageToSendboxItem(msg))
	}

	return &model.SendboxResult{
		Results: results,
		Count:   total,
	}, nil
}

func (s *GmailSender) DeleteSent(id int) error {
	msgID := fmt.Sprintf("%d", id)
	_, err := s.srv.Users.Messages.Trash("me", msgID).Do()
	if err != nil {
		return fmt.Errorf("trash sent message: %w", err)
	}
	return nil
}

func (s *GmailSender) ClearSent() error {
	var allIDs []string
	pageToken := ""
	for {
		call := s.srv.Users.Messages.List("me").MaxResults(500).Q("in:sent")
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return fmt.Errorf("list sent messages for clear: %w", err)
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

	err := s.srv.Users.Messages.BatchDelete("me", &gmail.BatchDeleteMessagesRequest{
		Ids: allIDs,
	}).Do()
	if err != nil {
		return fmt.Errorf("batch delete sent messages: %w", err)
	}
	return nil
}

func (s *GmailSender) sendWithXOAUTH2(to, msg string) error {
	c, err := smtp.Dial("smtp.gmail.com:587")
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer c.Close()

	if err := c.StartTLS(nil); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}

	if err := c.Auth(&xoauth2Auth{username: s.fromEmail, accessToken: s.auth.AccessToken}); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := c.Mail(s.fromEmail); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return w.Close()
}

type xoauth2Auth struct {
	username, accessToken string
}

func (a *xoauth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	resp := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.username, a.accessToken)
	return "XOAUTH2", []byte(resp), nil
}

func (a *xoauth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("unexpected XOAUTH2 challenge")
	}
	return nil, nil
}

func messageToSendboxItem(msg *gmail.Message) *model.SendboxItem {
	headers := msg.Payload.Headers
	to := getHeader(headers, "To")
	subject := getHeader(headers, "Subject")
	createdAt := ""
	if msg.InternalDate > 0 {
		createdAt = time.UnixMilli(msg.InternalDate).Format(time.RFC3339)
	}
	id, err := strconv.ParseInt(msg.Id, 10, 64)
	if err != nil {
		id = 0
	}
	return &model.SendboxItem{
		ID:        int(id),
		To:        to,
		Subject:   subject,
		CreatedAt: createdAt,
	}
}
