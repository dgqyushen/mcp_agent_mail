package service

import "agent-mail/model"

type WebhookService struct {
	mailboxSvc *MailboxService
}

func NewWebhookService(ms *MailboxService) *WebhookService {
	return &WebhookService{mailboxSvc: ms}
}

func (s *WebhookService) Get(userID int, alias string) (*model.WebhookSettings, error) {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return nil, err
	}
	return r.GetWebhook()
}

func (s *WebhookService) Set(userID int, alias string, cfg *model.WebhookSettings) error {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return err
	}
	return r.SetWebhook(cfg)
}
