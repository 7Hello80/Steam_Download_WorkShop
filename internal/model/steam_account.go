package model

import "time"

type SteamAccount struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	SteamUsername     string    `json:"steam_username"`
	EncryptedPassword string    `json:"-"` // never serialize
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type SaveSteamCredentialsRequest struct {
	SteamUsername string `json:"steam_username"`
	SteamPassword string `json:"steam_password"`
}
