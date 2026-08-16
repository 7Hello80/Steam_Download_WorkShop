package service

import (
	"database/sql"
	"fmt"
	"time"

	"steam-download-tool/internal/model"

	"github.com/google/uuid"
)

type SponsorService struct {
	db *sql.DB
}

func NewSponsorService(db *sql.DB) *SponsorService {
	return &SponsorService{db: db}
}

func (s *SponsorService) Create(req *model.CreateSponsorRequest) (*model.Sponsor, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Method != "wechat" && req.Method != "alipay" {
		return nil, fmt.Errorf("method must be 'wechat' or 'alipay'")
	}

	sp := &model.Sponsor{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Method:    req.Method,
		Amount:    req.Amount,
		Message:   req.Message,
		IsVisible: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := s.db.Exec(
		"INSERT INTO sponsors (id, name, method, amount, message, is_visible, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 1, ?, ?)",
		sp.ID, sp.Name, sp.Method, sp.Amount, sp.Message, sp.CreatedAt, sp.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create sponsor: %w", err)
	}

	return sp, nil
}

func (s *SponsorService) ListAll() ([]*model.Sponsor, error) {
	rows, err := s.db.Query(
		"SELECT id, name, method, amount, message, is_visible, created_at, updated_at FROM sponsors ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query sponsors: %w", err)
	}
	defer rows.Close()

	var results []*model.Sponsor
	for rows.Next() {
		sp := &model.Sponsor{}
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.Method, &sp.Amount, &sp.Message, &sp.IsVisible, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sponsor: %w", err)
		}
		results = append(results, sp)
	}
	if results == nil {
		results = []*model.Sponsor{}
	}
	return results, nil
}

func (s *SponsorService) ListVisible() ([]*model.Sponsor, error) {
	rows, err := s.db.Query(
		"SELECT id, name, method, amount, message, is_visible, created_at, updated_at FROM sponsors WHERE is_visible = 1 ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query visible sponsors: %w", err)
	}
	defer rows.Close()

	var results []*model.Sponsor
	for rows.Next() {
		sp := &model.Sponsor{}
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.Method, &sp.Amount, &sp.Message, &sp.IsVisible, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sponsor: %w", err)
		}
		results = append(results, sp)
	}
	if results == nil {
		results = []*model.Sponsor{}
	}
	return results, nil
}

func (s *SponsorService) GetByID(id string) (*model.Sponsor, error) {
	sp := &model.Sponsor{}
	err := s.db.QueryRow(
		"SELECT id, name, method, amount, message, is_visible, created_at, updated_at FROM sponsors WHERE id = ?",
		id,
	).Scan(&sp.ID, &sp.Name, &sp.Method, &sp.Amount, &sp.Message, &sp.IsVisible, &sp.CreatedAt, &sp.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sponsor not found")
		}
		return nil, fmt.Errorf("get sponsor: %w", err)
	}
	return sp, nil
}

func (s *SponsorService) Update(id string, req *model.UpdateSponsorRequest) (*model.Sponsor, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Method != "wechat" && req.Method != "alipay" {
		return nil, fmt.Errorf("method must be 'wechat' or 'alipay'")
	}

	if req.IsVisible != nil {
		result, err := s.db.Exec(
			"UPDATE sponsors SET name = ?, method = ?, amount = ?, message = ?, is_visible = ?, updated_at = ? WHERE id = ?",
			req.Name, req.Method, req.Amount, req.Message, *req.IsVisible, time.Now(), id,
		)
		if err != nil {
			return nil, fmt.Errorf("update sponsor: %w", err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return nil, fmt.Errorf("sponsor not found")
		}
	} else {
		result, err := s.db.Exec(
			"UPDATE sponsors SET name = ?, method = ?, amount = ?, message = ?, updated_at = ? WHERE id = ?",
			req.Name, req.Method, req.Amount, req.Message, time.Now(), id,
		)
		if err != nil {
			return nil, fmt.Errorf("update sponsor: %w", err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return nil, fmt.Errorf("sponsor not found")
		}
	}

	return s.GetByID(id)
}

func (s *SponsorService) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM sponsors WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete sponsor: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("sponsor not found")
	}
	return nil
}
