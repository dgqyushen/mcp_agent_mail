package sqlite

import (
	"database/sql"
	"errors"
)

type Token struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	TokenHash string `json:"-"`
	Prefix    string `json:"prefix"`
	CreatedAt string `json:"created_at"`
	IsActive  bool   `json:"is_active"`
}

func (db *DB) InsertToken(userID int, tokenHash, prefix string) (*Token, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE tokens SET is_active = 0 WHERE user_id = ? AND is_active = 1", userID); err != nil {
		return nil, err
	}
	result, err := tx.Exec(
		"INSERT INTO tokens (user_id, token_hash, prefix) VALUES (?, ?, ?)",
		userID, tokenHash, prefix,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Token{ID: int(id), UserID: userID, TokenHash: tokenHash, Prefix: prefix, IsActive: true}, nil
}

func (db *DB) FindActiveToken(tokenHash string) (*Token, error) {
	var t Token
	err := db.conn.QueryRow(
		`SELECT id, user_id, token_hash, prefix, created_at, is_active
		 FROM tokens WHERE token_hash = ? AND is_active = 1`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Prefix, &t.CreatedAt, &t.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (db *DB) GetActiveToken(userID int) (*Token, error) {
	var t Token
	err := db.conn.QueryRow(
		`SELECT id, user_id, token_hash, prefix, created_at, is_active
		 FROM tokens WHERE user_id = ? AND is_active = 1`, userID,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Prefix, &t.CreatedAt, &t.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (db *DB) DeactivateTokens(userID int) error {
	_, err := db.conn.Exec("UPDATE tokens SET is_active = 0 WHERE user_id = ? AND is_active = 1", userID)
	return err
}
