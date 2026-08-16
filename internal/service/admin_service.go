package service

import (
	"database/sql"
	"fmt"
	"time"

	"steam-download-tool/internal/model"
)

type AdminService struct {
	db *sql.DB
}

func NewAdminService(db *sql.DB) *AdminService {
	return &AdminService{db: db}
}

// GetAllUsers returns all users ordered by creation time desc.
func (s *AdminService) GetAllUsers() ([]*model.User, error) {
	rows, err := s.db.Query(
		`SELECT id, email, username, COALESCE(password_hash,''), COALESCE(github_id,''), COALESCE(avatar_url,''), role, email_verified, banned, COALESCE(banned_at,''), created_at, updated_at
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var bannedAtStr string
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.GitHubID, &u.AvatarURL, &u.Role, &u.EmailVerified, &u.Banned, &bannedAtStr, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if bannedAtStr != "" {
			t, _ := time.Parse("2006-01-02 15:04:05", bannedAtStr)
			if !t.IsZero() {
				u.BannedAt = &t
			}
		}
		users = append(users, u)
	}
	if users == nil {
		users = []*model.User{}
	}
	return users, nil
}

// PaginatedUsersResponse holds paginated user list with total count.
type PaginatedUsersResponse struct {
	Users    []*model.User `json:"users"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// GetUsersPaginated returns users with pagination. If page <= 0, returns all users.
func (s *AdminService) GetUsersPaginated(page, pageSize int) (*PaginatedUsersResponse, error) {
	// Default: return all users when no pagination requested
	if page <= 0 || pageSize <= 0 {
		users, err := s.GetAllUsers()
		if err != nil {
			return nil, err
		}
		return &PaginatedUsersResponse{
			Users:    users,
			Total:    len(users),
			Page:     1,
			PageSize: len(users),
		}, nil
	}

	// Get total count
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	// Clamp page size
	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	rows, err := s.db.Query(
		`SELECT id, email, username, COALESCE(password_hash,''), COALESCE(github_id,''), COALESCE(avatar_url,''), role, email_verified, banned, COALESCE(banned_at,''), created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var bannedAtStr string
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.GitHubID, &u.AvatarURL, &u.Role, &u.EmailVerified, &u.Banned, &bannedAtStr, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if bannedAtStr != "" {
			t, _ := time.Parse("2006-01-02 15:04:05", bannedAtStr)
			if !t.IsZero() {
				u.BannedAt = &t
			}
		}
		users = append(users, u)
	}
	if users == nil {
		users = []*model.User{}
	}

	return &PaginatedUsersResponse{
		Users:    users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// UpdateUserRole updates a user's role. Returns an error if the user is not found.
func (s *AdminService) UpdateUserRole(userID, role string) (*model.User, error) {
	if role != "user" && role != "admin" {
		return nil, fmt.Errorf("invalid role: %s (must be 'user' or 'admin')", role)
	}

	result, err := s.db.Exec(
		"UPDATE users SET role = ?, updated_at = ? WHERE id = ?",
		role, time.Now(), userID,
	)
	if err != nil {
		return nil, fmt.Errorf("update user role: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("user not found")
	}

	// Fetch and return updated user
	user := &model.User{}
	var bannedAtStr string
	err = s.db.QueryRow(
		`SELECT id, email, username, COALESCE(password_hash,''), COALESCE(github_id,''), COALESCE(avatar_url,''), role, email_verified, banned, COALESCE(banned_at,''), created_at, updated_at
		 FROM users WHERE id = ?`, userID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.GitHubID, &user.AvatarURL, &user.Role, &user.EmailVerified, &user.Banned, &bannedAtStr, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("fetch updated user: %w", err)
	}
	if bannedAtStr != "" {
		t, _ := time.Parse("2006-01-02 15:04:05", bannedAtStr)
		if !t.IsZero() {
			user.BannedAt = &t
		}
	}
	return user, nil
}

// GetUserByID returns a single user by ID.
func (s *AdminService) GetUserByID(userID string) (*model.User, error) {
	user := &model.User{}
	var bannedAtStr string
	err := s.db.QueryRow(
		`SELECT id, email, username, COALESCE(password_hash,''), COALESCE(github_id,''), COALESCE(avatar_url,''), role, email_verified, banned, COALESCE(banned_at,''), created_at, updated_at
		 FROM users WHERE id = ?`, userID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.GitHubID, &user.AvatarURL, &user.Role, &user.EmailVerified, &user.Banned, &bannedAtStr, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	if bannedAtStr != "" {
		t, _ := time.Parse("2006-01-02 15:04:05", bannedAtStr)
		if !t.IsZero() {
			user.BannedAt = &t
		}
	}
	return user, nil
}

// BanUser bans a user by ID. Returns the updated user.
func (s *AdminService) BanUser(userID string) (*model.User, error) {
	result, err := s.db.Exec(
		"UPDATE users SET banned = 1, banned_at = ?, updated_at = ? WHERE id = ? AND banned = 0",
		time.Now(), time.Now(), userID,
	)
	if err != nil {
		return nil, fmt.Errorf("ban user: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("user not found or already banned")
	}
	return s.GetUserByID(userID)
}

// UnbanUser unbans a user by ID. Returns the updated user.
func (s *AdminService) UnbanUser(userID string) (*model.User, error) {
	result, err := s.db.Exec(
		"UPDATE users SET banned = 0, banned_at = NULL, updated_at = ? WHERE id = ? AND banned = 1",
		time.Now(), userID,
	)
	if err != nil {
		return nil, fmt.Errorf("unban user: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("user not found or not banned")
	}
	return s.GetUserByID(userID)
}

// DeleteUser deletes a user by ID.
func (s *AdminService) DeleteUser(userID string) error {
	result, err := s.db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// DashboardStats holds aggregated stats for the admin dashboard.
type DashboardStats struct {
	TotalUsers       int `json:"total_users"`
	TotalTasks       int `json:"total_tasks"`
	PendingTasks     int `json:"pending_tasks"`
	RunningTasks     int `json:"running_tasks"`
	CompletedTasks   int `json:"completed_tasks"`
	ExpiredTasks     int `json:"expired_tasks"`
	TotalFilesSize   int64 `json:"total_files_size"`
}

// GetDashboardStats returns aggregated statistics using a single query.
func (s *AdminService) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	err := s.db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM users) AS total_users,
			COUNT(*) AS total_tasks,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending_tasks,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) AS running_tasks,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed_tasks,
			SUM(CASE WHEN status = 'expired' THEN 1 ELSE 0 END) AS expired_tasks,
			COALESCE(SUM(file_size), 0) AS total_files_size
		FROM download_tasks
	`).Scan(
		&stats.TotalUsers,
		&stats.TotalTasks,
		&stats.PendingTasks,
		&stats.RunningTasks,
		&stats.CompletedTasks,
		&stats.ExpiredTasks,
		&stats.TotalFilesSize,
	)
	if err != nil {
		return nil, fmt.Errorf("get dashboard stats: %w", err)
	}

	return stats, nil
}
