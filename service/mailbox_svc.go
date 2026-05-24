package service

import (
	"encoding/json"
	"fmt"

	"agent-mail/model"
	"agent-mail/provider"
	"agent-mail/provider/cloudflare"
	"agent-mail/store/sqlite"
)

type ProviderFactory interface {
	NewProvider(record model.MailboxRecord) provider.EmailProvider
}

type DefaultProviderFactory struct{}

func (f *DefaultProviderFactory) NewProvider(record model.MailboxRecord) provider.EmailProvider {
	auth := make(map[string]string)
	json.Unmarshal([]byte(record.AuthData), &auth)
	switch record.ProviderType {
	case "cloudflare":
		return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"])
	default:
		return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"])
	}
}

type MailboxService struct {
	db      *sqlite.DB
	factory ProviderFactory
}

func NewMailboxService(db *sqlite.DB, factory ProviderFactory) *MailboxService {
	if factory == nil {
		factory = &DefaultProviderFactory{}
	}
	return &MailboxService{db: db, factory: factory}
}

func (s *MailboxService) Add(alias, name, providerType, baseURL, authData string) error {
	if providerType == "" {
		providerType = "cloudflare"
	}
	rec := model.MailboxRecord{
		Alias:        alias,
		Name:         name,
		ProviderType: providerType,
		BaseURL:      baseURL,
		AuthData:     authData,
	}
	if err := s.db.InsertMailbox(rec); err != nil {
		return fmt.Errorf("add mailbox: %w", err)
	}
	defAlias, _ := s.db.GetSetting("default_mailbox")
	if defAlias == "" {
		s.db.SetSetting("default_mailbox", alias)
	}
	return nil
}

func (s *MailboxService) Remove(alias string) error {
	if err := s.db.DeleteMailbox(alias); err != nil {
		return fmt.Errorf("remove mailbox: %w", err)
	}
	defAlias, _ := s.db.GetSetting("default_mailbox")
	if defAlias == alias {
		list, _ := s.db.ListMailboxes()
		if len(list) > 0 {
			s.db.SetSetting("default_mailbox", list[0].Alias)
		} else {
			s.db.SetSetting("default_mailbox", "")
		}
	}
	return nil
}

func (s *MailboxService) Switch(alias string) error {
	if _, err := s.db.GetMailbox(alias); err != nil {
		return fmt.Errorf("switch mailbox: %w", err)
	}
	return s.db.SetSetting("default_mailbox", alias)
}

func (s *MailboxService) Default() string {
	v, _ := s.db.GetSetting("default_mailbox")
	return v
}

func (s *MailboxService) List() ([]model.MailboxInfo, error) {
	records, err := s.db.ListMailboxes()
	if err != nil {
		return nil, err
	}
	infos := make([]model.MailboxInfo, len(records))
	for i, r := range records {
		p := s.factory.NewProvider(r)
		valid := p.Validate() == nil
		infos[i] = model.MailboxInfo{
			Alias:        r.Alias,
			Name:         r.Name,
			ProviderType: r.ProviderType,
			BaseURL:      r.BaseURL,
			Valid:        valid,
		}
	}
	return infos, nil
}

func (s *MailboxService) Validate(alias string) (*model.SettingsResponse, error) {
	rec, err := s.Resolve(alias)
	if err != nil {
		return nil, err
	}
	p := s.factory.NewProvider(*rec)
	return p.GetSettings()
}

func (s *MailboxService) Resolve(alias string) (*model.MailboxRecord, error) {
	if alias == "" {
		alias = s.Default()
	}
	return s.db.GetMailbox(alias)
}

func (s *MailboxService) Provider(alias string) (provider.EmailProvider, error) {
	rec, err := s.Resolve(alias)
	if err != nil {
		return nil, err
	}
	return s.factory.NewProvider(*rec), nil
}
