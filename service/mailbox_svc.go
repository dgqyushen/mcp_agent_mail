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

func (s *MailboxService) Add(userID int, alias, name, providerType, baseURL, authData string) error {
	if providerType == "" {
		providerType = "cloudflare"
	}
	rec := model.MailboxRecord{
		UserID:       userID,
		Alias:        alias,
		Name:         name,
		ProviderType: providerType,
		BaseURL:      baseURL,
		AuthData:     authData,
	}
	if err := s.db.InsertMailbox(rec); err != nil {
		return fmt.Errorf("add mailbox: %w", err)
	}
	dk := fmt.Sprintf("default_mailbox_%d", userID)
	defAlias, _ := s.db.GetSetting(dk)
	if defAlias == "" {
		s.db.SetSetting(dk, alias)
	}
	return nil
}

func (s *MailboxService) Remove(userID int, alias string) error {
	if err := s.db.DeleteMailbox(userID, alias); err != nil {
		return fmt.Errorf("remove mailbox: %w", err)
	}
	dk := fmt.Sprintf("default_mailbox_%d", userID)
	defAlias, _ := s.db.GetSetting(dk)
	if defAlias == alias {
		list, _ := s.db.ListMailboxes(userID)
		if len(list) > 0 {
			s.db.SetSetting(dk, list[0].Alias)
		} else {
			s.db.SetSetting(dk, "")
		}
	}
	return nil
}

func (s *MailboxService) Switch(userID int, alias string) error {
	if _, err := s.db.GetMailbox(userID, alias); err != nil {
		return fmt.Errorf("switch mailbox: %w", err)
	}
	return s.db.SetSetting(fmt.Sprintf("default_mailbox_%d", userID), alias)
}

func (s *MailboxService) Default(userID int) string {
	v, _ := s.db.GetSetting(fmt.Sprintf("default_mailbox_%d", userID))
	return v
}

func (s *MailboxService) List(userID int) ([]model.MailboxInfo, error) {
	records, err := s.db.ListMailboxes(userID)
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

func (s *MailboxService) Validate(userID int, alias string) (*model.SettingsResponse, error) {
	rec, err := s.Resolve(userID, alias)
	if err != nil {
		return nil, err
	}
	p, err := s.factory.NewProvider(*rec)
	if err != nil {
		return nil, err
	}
	return p.GetSettings()
}

func (s *MailboxService) Resolve(userID int, alias string) (*model.MailboxRecord, error) {
	if alias == "" {
		alias = s.Default(userID)
	}
	return s.db.GetMailbox(userID, alias)
}

func (s *MailboxService) Provider(userID int, alias string) (provider.EmailProvider, error) {
	rec, err := s.Resolve(userID, alias)
	if err != nil {
		return nil, err
	}
	return s.factory.NewProvider(*rec)
}
