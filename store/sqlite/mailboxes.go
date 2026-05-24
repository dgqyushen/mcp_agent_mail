package sqlite

import (
	"fmt"
	"strings"

	"agent-mail/model"
)

func (db *DB) InsertMailbox(m model.MailboxRecord) error {
	_, err := db.conn.Exec(
		`INSERT INTO mailboxes (alias, name, provider_type, base_url, auth_data)
		 VALUES (?, ?, ?, ?, ?)`,
		m.Alias, m.Name, m.ProviderType, m.BaseURL, m.AuthData,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return fmt.Errorf("mailbox %q already exists", m.Alias)
		}
		return err
	}
	return nil
}

func (db *DB) GetMailbox(alias string) (*model.MailboxRecord, error) {
	var m model.MailboxRecord
	err := db.conn.QueryRow(
		`SELECT alias, name, provider_type, base_url, auth_data FROM mailboxes WHERE alias = ?`,
		alias,
	).Scan(&m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData)
	if err != nil {
		return nil, fmt.Errorf("mailbox %q not found: %w", alias, err)
	}
	if m.ProviderType == "" {
		m.ProviderType = "cloudflare"
	}
	return &m, nil
}

func (db *DB) ListMailboxes() ([]model.MailboxRecord, error) {
	rows, err := db.conn.Query(
		`SELECT alias, name, provider_type, base_url, auth_data FROM mailboxes ORDER BY alias`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.MailboxRecord
	for rows.Next() {
		var m model.MailboxRecord
		if err := rows.Scan(&m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData); err != nil {
			return nil, err
		}
		if m.ProviderType == "" {
			m.ProviderType = "cloudflare"
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (db *DB) DeleteMailbox(alias string) error {
	result, err := db.conn.Exec("DELETE FROM mailboxes WHERE alias = ?", alias)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mailbox %q not found", alias)
	}
	return nil
}
