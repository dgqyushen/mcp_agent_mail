package service

import "agent-mail/model"

type SendService struct {
	mailboxSvc *MailboxService
}

func NewSendService(ms *MailboxService) *SendService {
	return &SendService{mailboxSvc: ms}
}

func (s *SendService) Send(userID int, alias string, body *model.SendMailBody) error {
	snd, err := s.mailboxSvc.Sender(userID, alias)
	if err != nil {
		return err
	}
	return snd.SendMail(body)
}

func (s *SendService) CheckBalance(userID int, alias string) (int, error) {
	snd, err := s.mailboxSvc.Sender(userID, alias)
	if err != nil {
		return 0, err
	}
	return snd.CheckSendBalance()
}

func (s *SendService) ListSent(userID int, alias string, limit, offset int) (*model.SendboxResult, error) {
	snd, err := s.mailboxSvc.Sender(userID, alias)
	if err != nil {
		return nil, err
	}
	return snd.ListSent(limit, offset)
}

func (s *SendService) DeleteSent(userID int, alias string, id int) error {
	snd, err := s.mailboxSvc.Sender(userID, alias)
	if err != nil {
		return err
	}
	return snd.DeleteSent(id)
}

func (s *SendService) ClearSent(userID int, alias string) error {
	snd, err := s.mailboxSvc.Sender(userID, alias)
	if err != nil {
		return err
	}
	return snd.ClearSent()
}
