package cloudflare

import (
	"encoding/json"
	"fmt"

	"agent-mail/model"
	"agent-mail/provider"
)

func init() {
	provider.RegisterProvider("cloudflare", NewProvider)
}

func NewProvider(record model.MailboxRecord) (*provider.MailProvider, error) {
	auth := make(map[string]string)
	if err := json.Unmarshal([]byte(record.AuthData), &auth); err != nil {
		return nil, fmt.Errorf("invalid auth data for %q: %w", record.Alias, err)
	}
	c := New(record.BaseURL, auth["jwt"], auth["site_password"])
	return &provider.MailProvider{Receiver: c, Sender: c}, nil
}
