package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"agent-mail/model"
)

func Load(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.Config{
				DefaultMailbox: "",
				Mailboxes:      make(map[string]model.MailboxConfig),
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Mailboxes == nil {
		cfg.Mailboxes = make(map[string]model.MailboxConfig)
	}
	return &cfg, nil
}

func Save(path string, cfg *model.Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
