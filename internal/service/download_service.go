package service

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"steam-download-tool/internal/model"
)

type DownloadService struct {
	db *sql.DB
}

func NewDownloadService(db *sql.DB) *DownloadService {
	return &DownloadService{db: db}
}

func generateTaskID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (s *DownloadService) CreateTask(userID string, req *model.StartDownloadRequest) (*model.DownloadTask, error) {
	taskID := generateTaskID()
	now := time.Now()

	// OutputDir is left empty here — it will be resolved by the queue processor
	// using the configured output_dir from config.yaml (default: ./output).
	task := &model.DownloadTask{
		ID:            taskID,
		UserID:        userID,
		AppID:         req.AppID,
		PubfileID:     req.PubfileID,
		SteamUsername: req.SteamUsername,
		Status:        model.StatusPending,
		OutputDir:     "", // resolved by queue using cfg.OutputDir
		LoginID:       rand.Intn(2147483647) + 1,
		CreatedAt:     now,
	}

	_, err := s.db.Exec(
		`INSERT INTO download_tasks (id, user_id, app_id, pubfile_id, steam_username, status, output_dir, login_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.UserID, task.AppID, task.PubfileID, task.SteamUsername,
		task.Status, task.OutputDir, task.LoginID, task.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	return task, nil
}

func (s *DownloadService) ListTasks(userID string, page, limit int, status string) ([]*model.DownloadTask, int, error) {
	offset := (page - 1) * limit

	var total int
	countQuery := "SELECT COUNT(*) FROM download_tasks WHERE user_id = ?"
	countArgs := []interface{}{userID}
	if status != "" {
		countQuery += " AND status = ?"
		countArgs = append(countArgs, status)
	}
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	query := `SELECT id, user_id, app_id, pubfile_id, steam_username, status,
		COALESCE(queue_position,0), COALESCE(output_dir,''), COALESCE(zip_path,''),
		COALESCE(zip_filename,''), COALESCE(error_message,''), COALESCE(file_size,0),
		created_at, started_at, completed_at, expires_at
		FROM download_tasks WHERE user_id = ?`
	args := []interface{}{userID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]*model.DownloadTask, 0)
	for rows.Next() {
		t := &model.DownloadTask{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.AppID, &t.PubfileID, &t.SteamUsername,
			&t.Status, &t.QueuePosition, &t.OutputDir, &t.ZipPath, &t.ZipFilename,
			&t.ErrorMessage, &t.FileSize, &t.CreatedAt, &t.StartedAt,
			&t.CompletedAt, &t.ExpiresAt); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}

	return tasks, total, nil
}

func (s *DownloadService) GetTask(userID, taskID string) (*model.DownloadTask, error) {
	t := &model.DownloadTask{}
	err := s.db.QueryRow(
		`SELECT id, user_id, app_id, pubfile_id, steam_username, status,
		COALESCE(queue_position,0), COALESCE(output_dir,''), COALESCE(zip_path,''),
		COALESCE(zip_filename,''), COALESCE(error_message,''), COALESCE(file_size,0),
		created_at, started_at, completed_at, expires_at
		FROM download_tasks WHERE id = ? AND user_id = ?`,
		taskID, userID,
	).Scan(&t.ID, &t.UserID, &t.AppID, &t.PubfileID, &t.SteamUsername,
		&t.Status, &t.QueuePosition, &t.OutputDir, &t.ZipPath, &t.ZipFilename,
		&t.ErrorMessage, &t.FileSize, &t.CreatedAt, &t.StartedAt,
		&t.CompletedAt, &t.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *DownloadService) UpdateTaskStatus(userID, taskID, status string) error {
	result, err := s.db.Exec(
		`UPDATE download_tasks SET status = ? WHERE id = ? AND user_id = ?`,
		status, taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found or not owned by user")
	}
	return nil
}

func (s *DownloadService) ClearTaskFile(userID, taskID string) error {
	result, err := s.db.Exec(
		`UPDATE download_tasks SET zip_path = '', zip_filename = '', file_size = 0 WHERE id = ? AND user_id = ?`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("clear task file: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found or not owned by user")
	}
	return nil
}

func (s *DownloadService) GetTaskByID(taskID string) (*model.DownloadTask, error) {
	t := &model.DownloadTask{}
	err := s.db.QueryRow(
		`SELECT id, user_id, app_id, pubfile_id, steam_username, status,
		COALESCE(queue_position,0), COALESCE(output_dir,''), COALESCE(zip_path,''),
		COALESCE(zip_filename,''), COALESCE(error_message,''), COALESCE(file_size,0),
		created_at, started_at, completed_at, expires_at
		FROM download_tasks WHERE id = ?`,
		taskID,
	).Scan(&t.ID, &t.UserID, &t.AppID, &t.PubfileID, &t.SteamUsername,
		&t.Status, &t.QueuePosition, &t.OutputDir, &t.ZipPath, &t.ZipFilename,
		&t.ErrorMessage, &t.FileSize, &t.CreatedAt, &t.StartedAt,
		&t.CompletedAt, &t.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}
