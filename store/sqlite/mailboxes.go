package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"agent-mail/model"
	"agent-mail/provider"
)

func (db *DB) InsertMailbox(m model.MailboxRecord) error {
	_, err := db.conn.Exec(
		`INSERT INTO mailboxes (user_id, alias, name, provider_type, base_url, auth_data)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		m.UserID, m.Alias, m.Name, m.ProviderType, m.BaseURL, m.AuthData,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return fmt.Errorf("mailbox %q already exists", m.Alias)
		}
		return err
	}
	return nil
}

func (db *DB) GetMailbox(userID int, alias string) (*model.MailboxRecord, error) {
	var m model.MailboxRecord
	err := db.conn.QueryRow(
		`SELECT user_id, alias, name, provider_type, base_url, auth_data
		 FROM mailboxes WHERE user_id = ? AND alias = ?`,
		userID, alias,
	).Scan(&m.UserID, &m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("mailbox %q not found", alias)
		}
		return nil, err
	}
	if m.ProviderType == "" {
		registered := provider.RegisteredProviders()
		if len(registered) > 0 {
			m.ProviderType = registered[0]
		}
	}
	return &m, nil
}

func (db *DB) ListMailboxes(userID int) ([]model.MailboxRecord, error) {
	rows, err := db.conn.Query(
		`SELECT user_id, alias, name, provider_type, base_url, auth_data
		 FROM mailboxes WHERE user_id = ? ORDER BY alias`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.MailboxRecord
	var registered []string
	for rows.Next() {
		var m model.MailboxRecord
		if err := rows.Scan(&m.UserID, &m.Alias, &m.Name, &m.ProviderType, &m.BaseURL, &m.AuthData); err != nil {
			return nil, err
		}
		if m.ProviderType == "" {
			if registered == nil {
				registered = provider.RegisteredProviders()
			}
			if len(registered) > 0 {
				m.ProviderType = registered[0]
			}
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (db *DB) UpdateMailbox(m model.MailboxRecord) error {
	result, err := db.conn.Exec(
		`UPDATE mailboxes SET name=?, provider_type=?, base_url=?, auth_data=? WHERE user_id=? AND alias=?`,
		m.Name, m.ProviderType, m.BaseURL, m.AuthData, m.UserID, m.Alias,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("mailbox %q not found", m.Alias)
	}
	return nil
}

func (db *DB) DeleteMailbox(userID int, alias string) error {
	result, err := db.conn.Exec(
		"DELETE FROM mailboxes WHERE user_id = ? AND alias = ?", userID, alias,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("mailbox %q not found", alias)
	}
	return nil
}
