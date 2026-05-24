package model

import (
	"os"
	"path/filepath"
)

type MailboxConfig struct {
	Name         string `json:"name" toml:"name"`
	BaseURL      string `json:"base_url" toml:"base_url"`
	JWT          string `json:"jwt" toml:"jwt"`
	SitePassword string `json:"site_password" toml:"site_password"`
}

type Config struct {
	DefaultMailbox string                   `json:"default_mailbox" toml:"default_mailbox"`
	Mailboxes      map[string]MailboxConfig `json:"mailboxes" toml:"mailboxes"`
}

type SettingsResponse struct {
	Address     string `json:"address"`
	SendBalance int    `json:"send_balance"`
}

type ParsedMail struct {
	ID          int          `json:"id"`
	MessageID   string       `json:"message_id"`
	Source      string       `json:"source"`
	To          string       `json:"to"`
	CreatedAt   string       `json:"created_at"`
	Sender      string       `json:"sender"`
	Subject     string       `json:"subject"`
	Text        string       `json:"text"`
	HTML        string       `json:"html"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	MimeType    string `json:"mimeType"`
	Disposition string `json:"disposition"`
	Size        int    `json:"size"`
}

type MailboxInfo struct {
	Alias   string `json:"alias"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Valid   bool   `json:"valid"`
}

type SendMailBody struct {
	FromName string `json:"from_name"`
	ToMail   string `json:"to_mail"`
	ToName   string `json:"to_name"`
	Subject  string `json:"subject"`
	Content  string `json:"content"`
	IsHTML   bool   `json:"is_html"`
}

type AutoReplyConfig struct {
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	SourcePrefix string `json:"source_prefix"`
	Message      string `json:"message"`
	Enabled      bool   `json:"enabled"`
}

type WebhookSettings struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type PaginatedResult struct {
	Results []ParsedMail `json:"results"`
	Count   int          `json:"count"`
}

type SendboxItem struct {
	ID        int    `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"created_at"`
}

type SendboxResult struct {
	Results []SendboxItem `json:"results"`
	Count   int           `json:"count"`
}

type AttachmentItem struct {
	Key string `json:"key"`
}

type AttachmentListResult struct {
	Results []AttachmentItem `json:"results"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type SentEmailSummary struct {
	ID        int    `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"created_at"`
}

type ConfigPathFunc func() string

var DefaultConfigPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agent-mail", "config.toml")
}
