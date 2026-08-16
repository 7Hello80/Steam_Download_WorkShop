package model

import "time"

type Announcement struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsActive  bool      `json:"is_active"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateAnnouncementRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateAnnouncementRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	IsActive *bool  `json:"is_active,omitempty"`
}
