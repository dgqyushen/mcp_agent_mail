package service

import "agent-mail/model"

type SendService struct {
	mailboxSvc *MailboxService
}

func NewSendService(ms *MailboxService) *SendService {
	return &SendService{mailboxSvc: ms}
}

func (s *SendService) Send(userID int, alias string, body *model.SendMailBody) error {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return err
	}
	return p.SendMail(body)
}

func (s *SendService) CheckBalance(userID int, alias string) (int, error) {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return 0, err
	}
	return p.CheckSendBalance()
}

func (s *SendService) ListSent(userID int, alias string, limit, offset int) (*model.SendboxResult, error) {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return nil, err
	}
	return p.ListSent(limit, offset)
}

func (s *SendService) DeleteSent(userID int, alias string, id int) error {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return err
	}
	return p.DeleteSent(id)
}

func (s *SendService) ClearSent(userID int, alias string) error {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return err
	}
	return p.ClearSent()
}
