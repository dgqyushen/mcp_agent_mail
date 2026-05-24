package provider

import (
	"fmt"

	"agent-mail/model"
)

type ProviderFactory interface {
	NewProvider(record model.MailboxRecord) (*MailProvider, error)
}

var providerRegistry = map[string]func(model.MailboxRecord) (*MailProvider, error){}

func RegisterProvider(name string, fn func(model.MailboxRecord) (*MailProvider, error)) {
	providerRegistry[name] = fn
}

type DefaultProviderFactory struct{}

func (f *DefaultProviderFactory) NewProvider(record model.MailboxRecord) (*MailProvider, error) {
	if fn, ok := providerRegistry[record.ProviderType]; ok {
		return fn(record)
	}
	return nil, fmt.Errorf("unsupported provider type: %q", record.ProviderType)
}
