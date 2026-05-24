package service

import (
	"fmt"

	"agent-mail/model"
	"agent-mail/provider"
	"agent-mail/store/sqlite"
)

type MailboxService struct {
	db      *sqlite.DB
	factory provider.ProviderFactory
}

func NewMailboxService(db *sqlite.DB, factory provider.ProviderFactory) *MailboxService {
	if factory == nil {
		factory = &provider.DefaultProviderFactory{}
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
		valid := false
		if p.Receiver != nil {
			valid = p.Receiver.Validate() == nil
		} else if p.Sender != nil {
			valid = p.Sender.Validate() == nil
		}
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
	r, err := s.Receiver(userID, alias)
	if err != nil {
		return nil, err
	}
	return r.GetSettings()
}

func (s *MailboxService) Resolve(userID int, alias string) (*model.MailboxRecord, error) {
	if alias == "" {
		alias = s.Default(userID)
	}
	return s.db.GetMailbox(userID, alias)
}

func (s *MailboxService) Provider(userID int, alias string) (*provider.MailProvider, error) {
	rec, err := s.Resolve(userID, alias)
	if err != nil {
		return nil, err
	}
	return s.factory.NewProvider(*rec)
}

func (s *MailboxService) Receiver(userID int, alias string) (provider.MailReceiver, error) {
	p, err := s.Provider(userID, alias)
	if err != nil {
		return nil, err
	}
	if p.Receiver == nil {
		return nil, provider.ErrCapNotSupported
	}
	return p.Receiver, nil
}

func (s *MailboxService) Sender(userID int, alias string) (provider.MailSender, error) {
	p, err := s.Provider(userID, alias)
	if err != nil {
		return nil, err
	}
	if p.Sender == nil {
		return nil, provider.ErrCapNotSupported
	}
	return p.Sender, nil
}
