package service

import "agent-mail/model"

type WebhookService struct {
	mailboxSvc *MailboxService
}

func NewWebhookService(ms *MailboxService) *WebhookService {
	return &WebhookService{mailboxSvc: ms}
}

func (s *WebhookService) Get(userID int, alias string) (*model.WebhookSettings, error) {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return nil, err
	}
	return p.GetWebhook()
}

func (s *WebhookService) Set(userID int, alias string, cfg *model.WebhookSettings) error {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return err
	}
	return p.SetWebhook(cfg)
}
