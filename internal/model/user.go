package model

import "time"

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	PasswordHash  string    `json:"-"`
	GitHubID      string    `json:"github_id,omitempty"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	Role          string     `json:"role"`
	EmailVerified bool       `json:"email_verified"`
	Banned        bool       `json:"banned"`
	BannedAt      *time.Time `json:"banned_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type RegisterResponse struct {
	Message string `json:"message"`
	Email   string `json:"email"`
}
