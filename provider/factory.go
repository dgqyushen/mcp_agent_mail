package provider

import (
	"encoding/json"
	"fmt"

	"agent-mail/model"
	"agent-mail/provider/cloudflare"
)

type ProviderFactory interface {
	NewProvider(record model.MailboxRecord) (*MailProvider, error)
}

type DefaultProviderFactory struct{}

func (f *DefaultProviderFactory) NewProvider(record model.MailboxRecord) (*MailProvider, error) {
	auth := make(map[string]string)
	if err := json.Unmarshal([]byte(record.AuthData), &auth); err != nil {
		return nil, fmt.Errorf("invalid auth data for %q: %w", record.Alias, err)
	}
	switch record.ProviderType {
	case "cloudflare":
		c := cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"])
		return &MailProvider{Receiver: c, Sender: c}, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %q", record.ProviderType)
	}
}
