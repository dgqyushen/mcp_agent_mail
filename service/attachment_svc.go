package service

import "agent-mail/model"

type AttachmentService struct {
	mailboxSvc *MailboxService
}

func NewAttachmentService(ms *MailboxService) *AttachmentService {
	return &AttachmentService{mailboxSvc: ms}
}

func (s *AttachmentService) List(userID int, alias string) (*model.AttachmentListResult, error) {
	r, err := s.mailboxSvc.Receiver(userID, alias)
	if err != nil {
		return nil, err
	}
	return r.ListAttachments()
}
