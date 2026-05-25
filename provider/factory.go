package provider

import (
	"fmt"
	"sort"

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

func copyFields(fields []FieldDef) []FieldDef {
	out := make([]FieldDef, len(fields))
	copy(out, fields)
	return out
}

var providerFormInfos = map[string]ProviderFormInfo{}

// RegisterProviderFormInfo registers a provider's form configuration.
func RegisterProviderFormInfo(info ProviderFormInfo) {
	providerFormInfos[info.Type] = ProviderFormInfo{
		Type:   info.Type,
		Label:  info.Label,
		Fields: copyFields(info.Fields),
	}
}

// GetProviderFormInfos returns all registered provider form infos, sorted by Type.
func GetProviderFormInfos() []ProviderFormInfo {
	infos := make([]ProviderFormInfo, 0, len(providerFormInfos))
	for _, info := range providerFormInfos {
		infos = append(infos, ProviderFormInfo{
			Type:   info.Type,
			Label:  info.Label,
			Fields: copyFields(info.Fields),
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Type < infos[j].Type
	})
	return infos
}

// GetProviderFormInfo returns the form info for a specific provider type.
func GetProviderFormInfo(providerType string) *ProviderFormInfo {
	info, ok := providerFormInfos[providerType]
	if !ok {
		return nil
	}
	return &ProviderFormInfo{
		Type:   info.Type,
		Label:  info.Label,
		Fields: copyFields(info.Fields),
	}
}

func IsRegistered(name string) bool {
	_, ok := providerRegistry[name]
	return ok
}

func RegisteredProviders() []string {
	names := make([]string, 0, len(providerRegistry))
	for name := range providerRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
