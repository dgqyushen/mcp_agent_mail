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

// FieldDef describes a single form field for a provider's configuration UI.
type FieldDef struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Section  string `json:"section"`
	Required bool   `json:"required"`
}

// ProviderFormInfo describes the form fields needed to configure a provider.
type ProviderFormInfo struct {
	Type   string     `json:"type"`
	Label  string     `json:"label"`
	Fields []FieldDef `json:"fields"`
}

var providerFormInfos = map[string]ProviderFormInfo{}

func RegisterProviderFormInfo(info ProviderFormInfo) {
	providerFormInfos[info.Type] = info
}

func GetProviderFormInfos() []ProviderFormInfo {
	infos := make([]ProviderFormInfo, 0, len(providerFormInfos))
	for _, info := range providerFormInfos {
		infos = append(infos, info)
	}
	return infos
}

func GetProviderFormInfo(providerType string) *ProviderFormInfo {
	info, ok := providerFormInfos[providerType]
	if !ok {
		return nil
	}
	return &info
}
