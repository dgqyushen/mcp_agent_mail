package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"agent-mail/store/sqlite"
)

type UserService struct {
	db *sqlite.DB
}

func NewUserService(db *sqlite.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) CreateUser(name string) (*sqlite.User, string, error) {
	u, err := s.db.CreateUser(name)
	if err != nil {
		return nil, "", err
	}
	token, err := s.generateToken(u.ID)
	if err != nil {
		s.db.DeleteUser(u.ID)
		return nil, "", fmt.Errorf("create user failed: %w", err)
	}
	return u, token, nil
}

func (s *UserService) RefreshToken(userID int) (string, error) {
	return s.generateToken(userID)
}

func (s *UserService) ValidateToken(token string) (int, error) {
	hash := sha256Hex(token)
	t, err := s.db.FindActiveToken(hash)
	if err != nil {
		return 0, err
	}
	if t == nil {
		return 0, fmt.Errorf("invalid or expired token")
	}
	return t.UserID, nil
}

func (s *UserService) GetActiveTokenPrefix(userID int) string {
	t, err := s.db.GetActiveToken(userID)
	if err != nil || t == nil {
		return ""
	}
	return t.Prefix
}

func (s *UserService) ListUsers() ([]sqlite.User, error) {
	return s.db.ListUsers()
}

func (s *UserService) GetUser(id int) (*sqlite.User, error) {
	return s.db.GetUser(id)
}

func (s *UserService) DeleteUser(id int) error {
	if err := s.db.DeleteTokensByUser(id); err != nil {
		return err
	}
	return s.db.DeleteUser(id)
}

func (s *UserService) generateToken(userID int) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := "atm-" + hex.EncodeToString(b)
	prefix := token[:11] + "***"
	hash := sha256Hex(token)
	_, err := s.db.InsertToken(userID, hash, prefix)
	return token, err
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
