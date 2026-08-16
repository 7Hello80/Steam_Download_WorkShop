package handler

import (
	"encoding/json"
	"net/http"

	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/model"
	"steam-download-tool/internal/service"

	"github.com/go-chi/chi/v5"
)

type AnnouncementHandler struct {
	svc *service.AnnouncementService
}

func NewAnnouncementHandler(svc *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{svc: svc}
}

// ListActive returns all active announcements (for the banner).
func (h *AnnouncementHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	announcements, err := h.svc.ListActive()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, announcements)
}

// ListAll returns all announcements (admin).
func (h *AnnouncementHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	announcements, err := h.svc.ListAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, announcements)
}

// Create creates a new announcement (admin).
func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	createdBy := middleware.GetUserID(r)
	announcement, err := h.svc.Create(req.Title, req.Content, createdBy)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, announcement)
}

// Update updates an announcement (admin).
func (h *AnnouncementHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req model.UpdateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	announcement, err := h.svc.Update(id, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, announcement)
}

// Delete deletes an announcement (admin).
func (h *AnnouncementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "announcement deleted"})
}
