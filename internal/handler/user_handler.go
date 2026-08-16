package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/model"
	"steam-download-tool/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UserHandler struct {
	authSvc *service.AuthService
	userSvc *service.UserService
}

func NewUserHandler(authSvc *service.AuthService, userSvc *service.UserService) *UserHandler {
	return &UserHandler{authSvc: authSvc, userSvc: userSvc}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	user, err := h.authSvc.GetProfile(userID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	var req model.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authSvc.UpdateProfile(userID, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	// Limit upload size to 2MB
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)

	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large (max 2MB)")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".webp" {
		writeError(w, http.StatusBadRequest, "invalid file type: only png, jpg, jpeg, gif, webp are allowed")
		return
	}

	// Create avatars directory
	avatarsDir := filepath.Join(".", "file", "avatar")
	if err := os.MkdirAll(avatarsDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create avatars directory")
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filePath := filepath.Join(avatarsDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write avatar")
		return
	}

	// Construct avatar URL
	avatarURL := fmt.Sprintf("/file/avatar/%s", filename)

	// Update user's avatar URL in database
	user, err := h.authSvc.UpdateAvatar(userID, avatarURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) SaveSteamCredentials(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	var req model.SaveSteamCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.userSvc.SaveSteamCredentials(userID, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

func (h *UserHandler) GetSteamCredentials(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	accounts, err := h.userSvc.GetSteamCredentials(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if accounts == nil {
		accounts = []*model.SteamAccount{}
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (h *UserHandler) DeleteSteamCredentials(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	accountID := chi.URLParam(r, "id")
	if err := h.userSvc.DeleteSteamCredentials(userID, accountID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
