package service

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"agent-mail/model"
	"agent-mail/provider"
	"agent-mail/provider/cloudflare"
	"agent-mail/store/sqlite"
)

type ProviderFactory interface {
	NewProvider(record model.MailboxRecord) (provider.EmailProvider, error)
}

type DefaultProviderFactory struct{}

func (f *DefaultProviderFactory) NewProvider(record model.MailboxRecord) (provider.EmailProvider, error) {
	auth := make(map[string]string)
	if err := json.Unmarshal([]byte(record.AuthData), &auth); err != nil {
		return nil, fmt.Errorf("invalid auth data for %q: %w", record.Alias, err)
	}
	switch record.ProviderType {
	case "cloudflare":
		return cloudflare.New(record.BaseURL, auth["jwt"], auth["site_password"]), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %q", record.ProviderType)
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
	defAlias, err := s.db.GetSetting("default_mailbox")
	if err != nil {
		slog.Warn("failed to read default mailbox setting", "error", err)
	} else if defAlias == "" {
		if err := s.db.SetSetting("default_mailbox", alias); err != nil {
			slog.Warn("failed to set default mailbox", "alias", alias, "error", err)
		}
	}
	return nil
}

func (s *MailboxService) Remove(alias string) error {
	if err := s.db.DeleteMailbox(alias); err != nil {
		return fmt.Errorf("remove mailbox: %w", err)
	}
	defAlias, err := s.db.GetSetting("default_mailbox")
	if err != nil {
		slog.Warn("failed to read default mailbox", "error", err)
		return nil
	}
	if defAlias == alias {
		list, err := s.db.ListMailboxes()
		if err != nil {
			slog.Warn("failed to list mailboxes for fallback", "error", err)
			return nil
		}
		if len(list) > 0 {
			if err := s.db.SetSetting("default_mailbox", list[0].Alias); err != nil {
				slog.Warn("failed to set fallback default mailbox", "alias", list[0].Alias, "error", err)
			}
		} else {
			if err := s.db.SetSetting("default_mailbox", ""); err != nil {
				slog.Warn("failed to clear default mailbox", "error", err)
			}
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
		p, err := s.factory.NewProvider(r)
		if err != nil {
			return nil, err
		}
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
	p, err := s.factory.NewProvider(*rec)
	if err != nil {
		return nil, err
	}
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
	return s.factory.NewProvider(*rec)
}
