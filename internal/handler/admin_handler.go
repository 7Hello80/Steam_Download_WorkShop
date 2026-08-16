package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/service"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	svc *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListUsers returns all users with optional pagination.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	result, err := h.svc.GetUsersPaginated(page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// UpdateUserRole changes a user's role.
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.UpdateUserRole(userID, req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// GetDashboard returns aggregated statistics.
func (h *AdminHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetDashboardStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GetUser returns a single user's details.
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	user, err := h.svc.GetUserByID(userID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// BanUser bans a user.
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// Prevent banning yourself
	if userID == middleware.GetUserID(r) {
		writeError(w, http.StatusBadRequest, "不能封禁自己")
		return
	}

	user, err := h.svc.BanUser(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// UnbanUser unbans a user.
func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	user, err := h.svc.UnbanUser(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// DeleteUser deletes a user.
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// Prevent deleting yourself
	if userID == middleware.GetUserID(r) {
		writeError(w, http.StatusBadRequest, "不能删除自己")
		return
	}

	if err := h.svc.DeleteUser(userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "用户已删除"})
}
