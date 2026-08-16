package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"steam-download-tool/internal/config"
	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/service"

	"github.com/go-chi/chi/v5"
)

type FileHandler struct {
	dlSvc *service.DownloadService
	cfg   *config.Config
}

func NewFileHandler(dlSvc *service.DownloadService, cfg *config.Config) *FileHandler {
	return &FileHandler{dlSvc: dlSvc, cfg: cfg}
}

func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Only list completed files
	tasks, total, err := h.dlSvc.ListTasks(userID, page, limit, "completed")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list files")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"files": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	taskID := chi.URLParam(r, "id")

	task, err := h.dlSvc.GetTask(userID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	_ = task

	// Delete zip file from disk
	zipPath := filepath.Join(h.cfg.StaticDir, taskID)
	if err := os.RemoveAll(zipPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete file")
		return
	}

	// Update database: mark as cancelled and clear file paths
	// so it no longer appears in the completed files list
	if err := h.dlSvc.ClearTaskFile(userID, taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	if err := h.dlSvc.UpdateTaskStatus(userID, taskID, "cancelled"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	// Support token from query param for new-window downloads (e.g. /api/files/{id}/download?token=...)
	// The JWT middleware already enforces auth. The query token is an alternative for
	// scenarios where Authorization header cannot be set (e.g. window.open).
	// We only use the middleware-authenticated userID here.

	taskID := chi.URLParam(r, "id")

	task, err := h.dlSvc.GetTask(userID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	if task.Status != "completed" || task.ZipPath == "" {
		writeError(w, http.StatusNotFound, "file not available")
		return
	}

	// Check if file exists
	if _, err := os.Stat(task.ZipPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "file has expired and been deleted")
		return
	}

	// Open the file for serving
	file, err := os.Open(task.ZipPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to open file")
		return
	}
	defer file.Close()

	// Get file modtime for ETag/Last-Modified support
	st, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to stat file")
		return
	}

	filename := task.ZipFilename
	if filename == "" {
		filename = fmt.Sprintf("workshop_%s.zip", taskID)
	}

	// Sanitize filename for Content-Disposition header
	safeFilename := strings.ReplaceAll(filename, `"`, `_`)
	safeFilename = strings.ReplaceAll(safeFilename, `\`, `_`)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeFilename))
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Accept-Ranges", "bytes")

	// http.ServeContent uses sendfile(2) on Linux for zero-copy transfer,
	// supports Range requests for resumable/parallel downloads (IDM, aria2c),
	// and handles If-Modified-Since / If-Range headers automatically.
	http.ServeContent(w, r, safeFilename, st.ModTime(), file)
}
