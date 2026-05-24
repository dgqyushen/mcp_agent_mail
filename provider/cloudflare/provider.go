package cloudflare

import (
	"encoding/json"
	"fmt"

	"agent-mail/model"
	"agent-mail/provider"
)

func init() {
	provider.RegisterProvider("cloudflare", NewProvider)
	provider.RegisterProviderFormInfo(provider.ProviderFormInfo{
		Type:  "cloudflare",
		Label: "Cloudflare",
		Fields: []provider.FieldDef{
			{Key: "base_url", Label: "API Base URL", Type: "text", Section: "base_url", Required: true},
			{Key: "jwt", Label: "JWT Token", Type: "password", Section: "auth_data", Required: true},
			{Key: "site_password", Label: "Site Password", Type: "password", Section: "auth_data", Required: false},
		},
	})
}

func NewProvider(record model.MailboxRecord) (*provider.MailProvider, error) {
	auth := make(map[string]string)
	if err := json.Unmarshal([]byte(record.AuthData), &auth); err != nil {
		return nil, fmt.Errorf("invalid auth data for %q: %w", record.Alias, err)
	}
	c := New(record.BaseURL, auth["jwt"], auth["site_password"])
	return &provider.MailProvider{Receiver: c, Sender: c}, nil
}
