package provider

import "agent-mail/model"

type MailSender interface {
	SendMail(body *model.SendMailBody) error
	CheckSendBalance() (int, error)
	ListSent(limit, offset int) (*model.SendboxResult, error)
	DeleteSent(id int) error
	ClearSent() error
	Validate() error
}
