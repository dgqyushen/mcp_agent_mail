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

func (s *EmailService) List(alias string, limit, offset int) (*model.PaginatedResult, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.ListEmails(limit, offset)
}

func (s *EmailService) Get(alias string, id int) (*model.ParsedMail, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.GetEmail(id)
}

func (s *EmailService) Delete(alias string, id int) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.DeleteEmail(id)
}

func (s *EmailService) Clear(alias string) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.ClearInbox()
}

func (s *EmailService) Search(alias, query string, limit int) (*model.PaginatedResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return &model.PaginatedResult{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	result, err := p.ListEmails(100, 0)
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
