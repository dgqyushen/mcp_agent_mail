package service

import "agent-mail/model"

type AutoReplyService struct {
	mailboxSvc *MailboxService
}

func NewAutoReplyService(ms *MailboxService) *AutoReplyService {
	return &AutoReplyService{mailboxSvc: ms}
}

func (s *AutoReplyService) Get(userID int, alias string) (*model.AutoReplyConfig, error) {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return nil, err
	}
	return r.GetAutoReply()
}

func (s *AutoReplyService) Set(userID int, alias string, cfg *model.AutoReplyConfig) error {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return err
	}
	return r.SetAutoReply(cfg)
}
