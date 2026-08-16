package service

import (
	"database/sql"
	"errors"
	"fmt"

	"steam-download-tool/internal/crypto"
	"steam-download-tool/internal/model"

	"github.com/google/uuid"
	"time"
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) SaveSteamCredentials(userID string, req *model.SaveSteamCredentialsRequest) (*model.SteamAccount, error) {
	if req.SteamUsername == "" || req.SteamPassword == "" {
		return nil, errors.New("steam username and password are required")
	}

	encrypted, err := crypto.Encrypt(req.SteamPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}

	// Check if already exists
	var existingID string
	err = s.db.QueryRow(
		"SELECT id FROM steam_accounts WHERE user_id = ? AND steam_username = ?",
		userID, req.SteamUsername,
	).Scan(&existingID)

	if err == nil {
		// Update existing
		_, err = s.db.Exec(
			"UPDATE steam_accounts SET encrypted_password = ?, updated_at = ? WHERE id = ?",
			encrypted, time.Now(), existingID,
		)
		if err != nil {
			return nil, fmt.Errorf("update steam account: %w", err)
		}
		return &model.SteamAccount{
			ID:            existingID,
			UserID:        userID,
			SteamUsername: req.SteamUsername,
			UpdatedAt:     time.Now(),
		}, nil
	}

	// Create new
	account := &model.SteamAccount{
		ID:                uuid.New().String(),
		UserID:            userID,
		SteamUsername:     req.SteamUsername,
		EncryptedPassword: encrypted,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	_, err = s.db.Exec(
		"INSERT INTO steam_accounts (id, user_id, steam_username, encrypted_password, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		account.ID, account.UserID, account.SteamUsername, account.EncryptedPassword, account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert steam account: %w", err)
	}

	return account, nil
}

func (s *UserService) GetSteamCredentials(userID string) ([]*model.SteamAccount, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, steam_username, created_at, updated_at FROM steam_accounts WHERE user_id = ? ORDER BY updated_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query steam accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*model.SteamAccount
	for rows.Next() {
		a := &model.SteamAccount{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.SteamUsername, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan steam account: %w", err)
		}
		accounts = append(accounts, a)
	}

	return accounts, nil
}

func (s *UserService) DeleteSteamCredentials(userID, accountID string) error {
	result, err := s.db.Exec(
		"DELETE FROM steam_accounts WHERE id = ? AND user_id = ?",
		accountID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete steam account: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return errors.New("steam account not found")
	}
	return nil
}

func (s *UserService) GetDecryptedPassword(userID, steamUsername string) (string, error) {
	var encrypted string
	err := s.db.QueryRow(
		"SELECT encrypted_password FROM steam_accounts WHERE user_id = ? AND steam_username = ?",
		userID, steamUsername,
	).Scan(&encrypted)
	if err == sql.ErrNoRows {
		return "", errors.New("steam account not found")
	}
	if err != nil {
		return "", fmt.Errorf("query steam account: %w", err)
	}

	return crypto.Decrypt(encrypted)
}
