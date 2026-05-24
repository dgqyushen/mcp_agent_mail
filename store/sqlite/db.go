package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)
	if err := os.Chmod(path, 0600); err != nil {
		// file doesn't exist yet, SQLite will create it; chmod after first query
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	os.Chmod(path, 0600)
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS tokens (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			token_hash TEXT NOT NULL UNIQUE,
			prefix     TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			is_active  INTEGER NOT NULL DEFAULT 1
		);

		CREATE TABLE IF NOT EXISTS mailboxes (
			alias          TEXT NOT NULL,
			user_id        INTEGER NOT NULL DEFAULT 0,
			name           TEXT NOT NULL,
			provider_type  TEXT NOT NULL DEFAULT 'cloudflare',
			base_url       TEXT NOT NULL,
			auth_data      TEXT NOT NULL DEFAULT '{}',
			created_at     TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at     TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (user_id, alias)
		);
	`)
	return err
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}
