package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"steam-download-tool/internal/config"
	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/model"
	"steam-download-tool/internal/queue"
	"steam-download-tool/internal/service"

	"github.com/go-chi/chi/v5"
)

type DownloadHandler struct {
	dlSvc   *service.DownloadService
	userSvc *service.UserService
	queue   *queue.Queue
	cfg     *config.Config
}

func NewDownloadHandler(dlSvc *service.DownloadService, userSvc *service.UserService, q *queue.Queue, cfg *config.Config) *DownloadHandler {
	return &DownloadHandler{dlSvc: dlSvc, userSvc: userSvc, queue: q, cfg: cfg}
}

func (h *DownloadHandler) Start(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req model.StartDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AppID == 0 || req.PubfileID == 0 {
		writeError(w, http.StatusBadRequest, "app_id and pubfile_id are required")
		return
	}
	if req.SteamUsername == "" || req.SteamPassword == "" {
		writeError(w, http.StatusBadRequest, "steam_username and steam_password are required")
		return
	}

	// Save Steam credentials
	_, err := h.userSvc.SaveSteamCredentials(userID, &model.SaveSteamCredentialsRequest{
		SteamUsername: req.SteamUsername,
		SteamPassword: req.SteamPassword,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save credentials")
		return
	}

	// Create task in DB
	task, err := h.dlSvc.CreateTask(userID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task: "+err.Error())
		return
	}

	// Enqueue
	position, err := h.queue.Enqueue(task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue task: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, model.StartDownloadResponse{
		TaskID:        task.ID,
		QueuePosition: position,
	})
}

func (h *DownloadHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	status := r.URL.Query().Get("status")

	tasks, total, err := h.dlSvc.ListTasks(userID, page, limit, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *DownloadHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	taskID := chi.URLParam(r, "id")

	task, err := h.dlSvc.GetTask(userID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *DownloadHandler) GetOutput(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	taskID := chi.URLParam(r, "id")

	// Verify ownership
	task, err := h.dlSvc.GetTask(userID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	// Build log path
	outputDir := task.OutputDir
	if outputDir == "" {
		outputDir = fmt.Sprintf("%s/%s", h.cfg.OutputDir, task.ID)
	}
	logPath := outputDir + "/terminal.log"

	f, err := os.Open(logPath)
	if err != nil {
		// No log yet — return empty
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"task_id": taskID,
			"lines":   []string{},
		})
		return
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read terminal log")
		return
	}

	// Split into lines, keep last 2000 for response size
	lines := strings.Split(string(content), "\n")
	// Remove trailing empty line from trailing \n
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > 2000 {
		lines = lines[len(lines)-2000:]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_id": taskID,
		"lines":   lines,
	})
}

func (h *DownloadHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	taskID := chi.URLParam(r, "id")

	// Verify ownership
	task, err := h.dlSvc.GetTask(userID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	_ = task

	if err := h.queue.Cancel(taskID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
