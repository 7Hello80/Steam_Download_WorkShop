package model

import "time"

const (
	StatusPending     = "pending"
	StatusQueued      = "queued"
	StatusDownloading = "downloading"
	StatusCompleted   = "completed"
	StatusFailed      = "failed"
	StatusCancelled   = "cancelled"
	StatusExpired     = "expired"
	StatusTimeout     = "timeout"
)

type DownloadTask struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	AppID          int64      `json:"app_id"`
	PubfileID      int64      `json:"pubfile_id"`
	SteamUsername  string     `json:"steam_username"`
	Status         string     `json:"status"`
	QueuePosition  int        `json:"queue_position"`
	OutputDir      string     `json:"output_dir,omitempty"`
	ZipPath        string     `json:"zip_path,omitempty"`
	ZipFilename    string     `json:"zip_filename,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	FileSize       int64      `json:"file_size"`
	LoginID        int        `json:"login_id"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

type StartDownloadRequest struct {
	AppID           int64  `json:"app_id"`
	PubfileID       int64  `json:"pubfile_id"`
	SteamUsername   string `json:"steam_username"`
	SteamPassword   string `json:"steam_password"`
	SaveCredentials bool   `json:"save_credentials"`
}

type StartDownloadResponse struct {
	TaskID        string `json:"task_id"`
	QueuePosition int    `json:"queue_position"`
}
