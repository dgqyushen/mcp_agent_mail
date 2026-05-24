package provider

import "agent-mail/model"

type EmailProvider interface {
	GetSettings() (*model.SettingsResponse, error)

	ListEmails(limit, offset int) (*model.PaginatedResult, error)
	GetEmail(id int) (*model.ParsedMail, error)
	DeleteEmail(id int) error
	ClearInbox() error

	SendMail(body *model.SendMailBody) error
	CheckSendBalance() (int, error)

	ListSent(limit, offset int) (*model.SendboxResult, error)
	DeleteSent(id int) error
	ClearSent() error

	GetAutoReply() (*model.AutoReplyConfig, error)
	SetAutoReply(cfg *model.AutoReplyConfig) error

	GetWebhook() (*model.WebhookSettings, error)
	SetWebhook(cfg *model.WebhookSettings) error

	ListAttachments() (*model.AttachmentListResult, error)

	Validate() error
}
