package service

import "agent-mail/model"

type AttachmentService struct {
	mailboxSvc *MailboxService
}

func NewAttachmentService(ms *MailboxService) *AttachmentService {
	return &AttachmentService{mailboxSvc: ms}
}

func (s *AttachmentService) List(userID int, alias string) (*model.AttachmentListResult, error) {
	p, err := s.mailboxSvc.Provider(userID, alias)
	if err != nil {
		return nil, err
	}
	return p.ListAttachments()
}
