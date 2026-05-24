package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) CreateUser(name string) (*User, error) {
	result, err := db.conn.Exec("INSERT INTO users (name) VALUES (?)", name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, fmt.Errorf("user %q already exists", name)
		}
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &User{ID: int(id), Name: name}, nil
}

func (db *DB) GetUser(id int) (*User, error) {
	var u User
	err := db.conn.QueryRow(
		"SELECT id, name, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user %d not found", id)
		}
		return nil, err
	}
	return &u, nil
}

func (db *DB) ListUsers() ([]User, error) {
	rows, err := db.conn.Query("SELECT id, name, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *DB) DeleteUser(id int) error {
	result, err := db.conn.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user %d not found", id)
	}
	return nil
}
