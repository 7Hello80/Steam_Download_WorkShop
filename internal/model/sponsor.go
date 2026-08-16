package model

import "time"

type Sponsor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Method    string    `json:"method"` // "wechat" or "alipay"
	Amount    string    `json:"amount"`
	Message   string    `json:"message"`
	IsVisible bool      `json:"is_visible"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateSponsorRequest struct {
	Name    string `json:"name"`
	Method  string `json:"method"`
	Amount  string `json:"amount"`
	Message string `json:"message"`
}

type UpdateSponsorRequest struct {
	Name      string `json:"name"`
	Method    string `json:"method"`
	Amount    string `json:"amount"`
	Message   string `json:"message"`
	IsVisible *bool  `json:"is_visible,omitempty"`
}
