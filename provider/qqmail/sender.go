package qqmail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"agent-mail/model"
	"agent-mail/provider"
)

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
	msg := provider.BuildSMTPMessage(body.FromName, s.auth.Username, body.ToMail, body.ToName, body.Subject, body.Content, body.IsHTML)
	return s.sendWithLOGIN(body.ToMail, msg)
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

func (s *QQmailSender) sendWithLOGIN(to, msg string) error {
	conn, err := tls.Dial("tcp", "smtp.qq.com:465", &tls.Config{ServerName: "smtp.qq.com"})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	c, err := smtp.NewClient(conn, "smtp.qq.com")
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Close()

	if err := c.Auth(&loginAuth{username: s.auth.Username, password: s.auth.Password}); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := c.Mail(s.auth.Username); err != nil {
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

type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	challenge := strings.TrimSpace(string(fromServer))
	switch {
	case strings.EqualFold(challenge, "Username:"):
		return []byte(a.username), nil
	case strings.EqualFold(challenge, "Password:"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected LOGIN challenge: %s", string(fromServer))
	}
}


