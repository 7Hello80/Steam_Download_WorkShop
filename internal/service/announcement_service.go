package service

import (
	"database/sql"
	"fmt"
	"time"

	"steam-download-tool/internal/model"

	"github.com/google/uuid"
)

type AnnouncementService struct {
	db *sql.DB
}

func NewAnnouncementService(db *sql.DB) *AnnouncementService {
	return &AnnouncementService{db: db}
}

func (s *AnnouncementService) Create(title, content, createdBy string) (*model.Announcement, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	a := &model.Announcement{
		ID:        uuid.New().String(),
		Title:     title,
		Content:   content,
		IsActive:  true,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := s.db.Exec(
		"INSERT INTO announcements (id, title, content, is_active, created_by, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?, ?)",
		a.ID, a.Title, a.Content, a.CreatedBy, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}

	return a, nil
}

func (s *AnnouncementService) ListAll() ([]*model.Announcement, error) {
	rows, err := s.db.Query(
		"SELECT id, title, content, is_active, created_by, created_at, updated_at FROM announcements ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query announcements: %w", err)
	}
	defer rows.Close()

	var results []*model.Announcement
	for rows.Next() {
		a := &model.Announcement{}
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.IsActive, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan announcement: %w", err)
		}
		results = append(results, a)
	}
	if results == nil {
		results = []*model.Announcement{}
	}
	return results, nil
}

func (s *AnnouncementService) ListActive() ([]*model.Announcement, error) {
	rows, err := s.db.Query(
		"SELECT id, title, content, is_active, created_by, created_at, updated_at FROM announcements WHERE is_active = 1 ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query active announcements: %w", err)
	}
	defer rows.Close()

	var results []*model.Announcement
	for rows.Next() {
		a := &model.Announcement{}
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.IsActive, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan announcement: %w", err)
		}
		results = append(results, a)
	}
	if results == nil {
		results = []*model.Announcement{}
	}
	return results, nil
}

func (s *AnnouncementService) GetByID(id string) (*model.Announcement, error) {
	a := &model.Announcement{}
	err := s.db.QueryRow(
		"SELECT id, title, content, is_active, created_by, created_at, updated_at FROM announcements WHERE id = ?",
		id,
	).Scan(&a.ID, &a.Title, &a.Content, &a.IsActive, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("announcement not found")
		}
		return nil, fmt.Errorf("get announcement: %w", err)
	}
	return a, nil
}

func (s *AnnouncementService) Update(id string, req *model.UpdateAnnouncementRequest) (*model.Announcement, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	// Build dynamic update — only set is_active when explicitly provided
	if req.IsActive != nil {
		result, err := s.db.Exec(
			"UPDATE announcements SET title = ?, content = ?, is_active = ?, updated_at = ? WHERE id = ?",
			req.Title, req.Content, *req.IsActive, time.Now(), id,
		)
		if err != nil {
			return nil, fmt.Errorf("update announcement: %w", err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return nil, fmt.Errorf("announcement not found")
		}
	} else {
		result, err := s.db.Exec(
			"UPDATE announcements SET title = ?, content = ?, updated_at = ? WHERE id = ?",
			req.Title, req.Content, time.Now(), id,
		)
		if err != nil {
			return nil, fmt.Errorf("update announcement: %w", err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return nil, fmt.Errorf("announcement not found")
		}
	}

	return s.GetByID(id)
}

func (s *AnnouncementService) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM announcements WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("announcement not found")
	}
	return nil
}
