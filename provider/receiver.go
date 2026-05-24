package provider

import "agent-mail/model"

type MailReceiver interface {
	GetSettings() (*model.SettingsResponse, error)
	ListEmails(limit, offset int) (*model.PaginatedResult, error)
	GetEmail(id int) (*model.ParsedMail, error)
	DeleteEmail(id int) error
	ClearInbox() error
	GetAutoReply() (*model.AutoReplyConfig, error)
	SetAutoReply(cfg *model.AutoReplyConfig) error
	GetWebhook() (*model.WebhookSettings, error)
	SetWebhook(cfg *model.WebhookSettings) error
	ListAttachments() (*model.AttachmentListResult, error)
	Validate() error
}
