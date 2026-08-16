package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"steam-download-tool/internal/config"
	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/mpkg"
	"steam-download-tool/internal/service"
	"steam-download-tool/internal/storage"

	"github.com/go-chi/chi/v5"
)

// ConvertHandler handles zip-to-mpkg conversion requests.
type ConvertHandler struct {
	dlSvc *service.DownloadService
	cfg   *config.Config
}

// NewConvertHandler creates a new ConvertHandler.
func NewConvertHandler(dlSvc *service.DownloadService, cfg *config.Config) *ConvertHandler {
	return &ConvertHandler{dlSvc: dlSvc, cfg: cfg}
}

// ConvertibleFile represents a completed download that can be converted to mpkg.
type ConvertibleFile struct {
	TaskID      string   `json:"task_id"`
	AppID       int64    `json:"app_id"`
	PubfileID   int64    `json:"pubfile_id"`
	Title       string   `json:"title"`
	VideoFiles  []string `json:"video_files"`
	PreviewName string   `json:"preview_name"`
	FileSize    int64    `json:"file_size"`
	Converted   bool     `json:"converted"`
	MpkgName    string   `json:"mpkg_name,omitempty"`
	MpkgSize    int64    `json:"mpkg_size,omitempty"`
	DownloadURL string   `json:"download_url,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
}

// ListConvertible returns all completed downloads that can be converted to mpkg.
func (h *ConvertHandler) ListConvertible(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	tasks, _, err := h.dlSvc.ListTasks(userID, 1, 200, "completed")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	var convertible []ConvertibleFile
	for _, t := range tasks {
		if t.ZipPath == "" {
			continue
		}
		if _, err := os.Stat(t.ZipPath); os.IsNotExist(err) {
			continue
		}

		// Try to analyze the zip for display info (best-effort)
		var title string
		videoFiles := make([]string, 0)
		var previewName string

		if info, err := mpkg.AnalyzeZip(t.ZipPath); err == nil {
			title = info.Title
			for _, v := range info.VideoFiles {
				videoFiles = append(videoFiles, filepath.Base(v))
			}
			previewName = filepath.Base(info.PreviewImage)
		}

		if title == "" {
			title = fmt.Sprintf("%d", t.PubfileID)
		}

		// Check if already converted (look for *.mpkg in task dir)
		taskDir := filepath.Join(h.cfg.StaticDir, t.ID)
		mpkgName, mpkgSize, converted := findExistingMpkg(taskDir)

		c := ConvertibleFile{
			TaskID:      t.ID,
			AppID:       t.AppID,
			PubfileID:   t.PubfileID,
			Title:       title,
			VideoFiles:  videoFiles,
			PreviewName: previewName,
			FileSize:    t.FileSize,
			Converted:   converted,
		}

		if converted {
			c.MpkgName = mpkgName
			c.MpkgSize = mpkgSize
			c.DownloadURL = fmt.Sprintf("/static/%s/%s",
				t.ID, mpkgName)
		}

		if t.ExpiresAt != nil {
			c.ExpiresAt = t.ExpiresAt.Format(time.RFC3339)
		}

		convertible = append(convertible, c)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"files": convertible,
		"total": len(convertible),
	})
}

// Convert performs the mpkg conversion for a task.
func (h *ConvertHandler) Convert(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req struct {
		TaskID string `json:"task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TaskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	task, err := h.dlSvc.GetTask(userID, req.TaskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	if task.Status != "completed" {
		writeError(w, http.StatusBadRequest, "task is not completed")
		return
	}

	if _, err := os.Stat(task.ZipPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "zip file not found on disk")
		return
	}

	// Check if already converted — return existing result directly
	taskDir := filepath.Join(h.cfg.StaticDir, req.TaskID)
	if mpkgName, mpkgSize, ok := findExistingMpkg(taskDir); ok {
		title := fmt.Sprintf("%d", task.PubfileID)
		if info, err := mpkg.AnalyzeZip(task.ZipPath); err == nil && info.Title != "" {
			title = info.Title
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":       "completed",
			"task_id":      req.TaskID,
			"title":        title,
			"filename":     mpkgName,
			"file_size":    mpkgSize,
			"download_url": fmt.Sprintf("/static/%s/%s", req.TaskID, mpkgName),
		})
		return
	}

	// Analyze zip to determine wallpaper type and get title
	info, analyzeErr := mpkg.AnalyzeZip(task.ZipPath)

	title := fmt.Sprintf("%d", task.PubfileID)
	if analyzeErr == nil && info.Title != "" {
		title = info.Title
	}

	// Detect scene type: not a video, and has scene.pkg
	isSceneType := analyzeErr == nil && !info.IsVideoType && info.HasScenePkg

	safeTitle := sanitizeFilename(title)
	if safeTitle == "" {
		safeTitle = fmt.Sprintf("file_%s", req.TaskID)
	}
	mpkgFilename := safeTitle + ".mpkg"

	mpkgPath := filepath.Join(taskDir, mpkgFilename)

	// Convert based on wallpaper type
	if isSceneType {
		if err := convertSceneZip(task.ZipPath, mpkgPath, task.PubfileID); err != nil {
			writeError(w, http.StatusInternalServerError, "scene conversion failed: "+err.Error())
			return
		}
	} else {
		if err := mpkg.ConvertZipToMPKG(task.ZipPath, mpkgPath); err != nil {
			writeError(w, http.StatusInternalServerError, "conversion failed: "+err.Error())
			return
		}
	}

	mpkgSize := int64(0)
	if st, err := os.Stat(mpkgPath); err == nil {
		mpkgSize = st.Size()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "completed",
		"task_id":      req.TaskID,
		"title":        title,
		"filename":     mpkgFilename,
		"file_size":    mpkgSize,
		"download_url": fmt.Sprintf("/static/%s/%s", req.TaskID, mpkgFilename),
	})
}

// Download serves the converted mpkg file.
func (h *ConvertHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	taskID := chi.URLParam(r, "id")
	filename := r.URL.Query().Get("file")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "file query param is required")
		return
	}

	_, err := h.dlSvc.GetTask(userID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	filename = filepath.Base(filename)
	if strings.Contains(filename, "..") {
		writeError(w, http.StatusBadRequest, "invalid filename")
		return
	}

	mpkgPath := filepath.Join(h.cfg.StaticDir, taskID, filename)

	if _, err := os.Stat(mpkgPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "mpkg file not found — it may have expired or been deleted")
		return
	}

	file, err := os.Open(mpkgPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to open file")
		return
	}
	defer file.Close()

	st, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to stat file")
		return
	}

	safeFilename := strings.ReplaceAll(filename, `"`, `_`)
	safeFilename = strings.ReplaceAll(safeFilename, `\`, `_`)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeFilename))
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Accept-Ranges", "bytes")

	// http.ServeContent uses sendfile(2) for zero-copy transfer on Linux,
	// supports Range requests (resumable/parallel downloads),
	// and handles ETag / If-Modified-Since automatically.
	http.ServeContent(w, r, safeFilename, st.ModTime(), file)
}

// convertSceneZip handles scene-type wallpaper conversion with scene.pkg unpacking.
// Steps:
//  1. Extract zip to temp directory
//  2. Rename scene.pkg to scene.mpkg (required for unpacking)
//  3. Unpack scene.mpkg (extracts scene.json, effects/, materials/, models/, shaders/)
//  4. Delete scene.mpkg
//  5. Post-process unpacked files for mobile compatibility
//  6. Re-zip temp directory
//  7. Convert new zip to mpkg
//  8. Clean up temp directory
func convertSceneZip(zipPath, mpkgPath string, pubfileID int64) error {
	tempDir, err := os.MkdirTemp("", "scene-convert-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Step 1: Extract zip to temp directory
	if err := storage.ExtractZip(zipPath, tempDir); err != nil {
		return fmt.Errorf("extract zip: %w", err)
	}

	// Step 2-4: Find scene.pkg, rename to scene.mpkg, unpack, delete
	scenePkgPath := filepath.Join(tempDir, "scene.pkg")
	sceneMpkgPath := filepath.Join(tempDir, "scene.mpkg")
	if _, err := os.Stat(scenePkgPath); err == nil {
		if err := os.Rename(scenePkgPath, sceneMpkgPath); err != nil {
			return fmt.Errorf("rename scene.pkg: %w", err)
		}

		if err := mpkg.UnpackMPKG(sceneMpkgPath, tempDir); err != nil {
			return fmt.Errorf("unpack scene.mpkg: %w", err)
		}

		if err := os.Remove(sceneMpkgPath); err != nil {
			return fmt.Errorf("remove scene.mpkg: %w", err)
		}
	}

	// Step 5: Post-process unpacked files for mobile compatibility
	if err := mpkg.ProcessSceneFiles(tempDir); err != nil {
		return fmt.Errorf("process scene files: %w", err)
	}

	// Step 5b: Remove terminal.log (DepotDownloader artifact, not needed in wallpaper)
	terminalLogPath := filepath.Join(tempDir, "terminal.log")
	os.Remove(terminalLogPath)

	// Step 5c: Add workshop URL to project.json for mobile compatibility
	if err := mpkg.PatchProjectJSON(tempDir, pubfileID); err != nil {
		return fmt.Errorf("patch project.json: %w", err)
	}

	// Step 6: Re-zip temp directory
	rezipPath := mpkgPath + ".temp.zip"
	if err := storage.ZipDirectory(tempDir, rezipPath); err != nil {
		return fmt.Errorf("re-zip: %w", err)
	}
	defer os.Remove(rezipPath)

	// Verify the temp zip is valid before converting
	if st, err := os.Stat(rezipPath); err != nil {
		return fmt.Errorf("re-zip stat: %w", err)
	} else if st.Size() == 0 {
		return fmt.Errorf("re-zip created empty file")
	}

	if err := mpkg.ConvertZipToMPKG(rezipPath, mpkgPath); err != nil {
		return fmt.Errorf("convert to mpkg: %w", err)
	}

	return nil
}

// findExistingMpkg looks for any .mpkg file in the task directory.
// Returns name, size, and whether one was found.
func findExistingMpkg(taskDir string) (string, int64, bool) {
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return "", 0, false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mpkg") {
			info, err := e.Info()
			if err != nil {
				continue
			}
			return e.Name(), info.Size(), true
		}
	}
	return "", 0, false
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", "\n", "", "\r", "", "\t", " ",
	)
	s = replacer.Replace(s)
	s = strings.TrimSpace(s)
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}
