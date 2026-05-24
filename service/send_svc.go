package service

import "agent-mail/model"

type SendService struct {
	mailboxSvc *MailboxService
}

func NewSendService(ms *MailboxService) *SendService {
	return &SendService{mailboxSvc: ms}
}

func (s *SendService) Send(alias string, body *model.SendMailBody) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.SendMail(body)
}

func (s *SendService) CheckBalance(alias string) (int, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return 0, err
	}
	return p.CheckSendBalance()
}

func (s *SendService) ListSent(alias string, limit, offset int) (*model.SendboxResult, error) {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return nil, err
	}
	return p.ListSent(limit, offset)
}

func (s *SendService) DeleteSent(alias string, id int) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.DeleteSent(id)
}

func (s *SendService) ClearSent(alias string) error {
	p, err := s.mailboxSvc.Provider(alias)
	if err != nil {
		return err
	}
	return p.ClearSent()
}
