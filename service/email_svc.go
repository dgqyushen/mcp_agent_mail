package service

import (
	"fmt"
	"strings"

	"agent-mail/model"
)

type EmailService struct {
	mailboxSvc *MailboxService
}

func NewEmailService(ms *MailboxService) *EmailService {
	return &EmailService{mailboxSvc: ms}
}

func (s *EmailService) List(userID int, alias string, limit, offset int) (*model.PaginatedResult, error) {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return nil, err
	}
	return r.ListEmails(limit, offset)
}

func (s *EmailService) Get(userID int, alias string, id int) (*model.ParsedMail, error) {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return nil, err
	}
	return r.GetEmail(id)
}

func (s *EmailService) Delete(userID int, alias string, id int) error {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return err
	}
	return r.DeleteEmail(id)
}

func (s *EmailService) Clear(userID int, alias string) error {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return err
	}
	return r.ClearInbox()
}

func (s *EmailService) Search(userID int, alias, query string, limit int) (*model.PaginatedResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return &model.PaginatedResult{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return nil, err
	}
	result, err := r.ListEmails(100, 0)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	q = strings.ToLower(q)
	var filtered []model.ParsedMail
	for _, m := range result.Results {
		if strings.Contains(strings.ToLower(m.Sender), q) ||
			strings.Contains(strings.ToLower(m.Subject), q) {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return &model.PaginatedResult{Results: filtered, Count: len(filtered)}, nil
}
