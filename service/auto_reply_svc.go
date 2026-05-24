package service

import "agent-mail/model"

type AutoReplyService struct {
	mailboxSvc *MailboxService
}

func NewAutoReplyService(ms *MailboxService) *AutoReplyService {
	return &AutoReplyService{mailboxSvc: ms}
}

func (s *AutoReplyService) Get(alias string) (*model.AutoReplyConfig, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.GetAutoReply()
}

func (s *AutoReplyService) Set(alias string, cfg *model.AutoReplyConfig) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.SetAutoReply(cfg)
}
