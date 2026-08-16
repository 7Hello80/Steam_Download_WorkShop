package handler

import (
	"encoding/json"
	"net/http"

	"steam-download-tool/internal/model"
	"steam-download-tool/internal/service"

	"github.com/go-chi/chi/v5"
)

type SponsorHandler struct {
	svc *service.SponsorService
}

func NewSponsorHandler(svc *service.SponsorService) *SponsorHandler {
	return &SponsorHandler{svc: svc}
}

// ListVisible returns all visible sponsors (public).
func (h *SponsorHandler) ListVisible(w http.ResponseWriter, r *http.Request) {
	sponsors, err := h.svc.ListVisible()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sponsors)
}

// ListAll returns all sponsors (admin).
func (h *SponsorHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	sponsors, err := h.svc.ListAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sponsors)
}

// Create creates a new sponsor (admin).
func (h *SponsorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSponsorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sponsor, err := h.svc.Create(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, sponsor)
}

// Update updates a sponsor (admin).
func (h *SponsorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req model.UpdateSponsorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sponsor, err := h.svc.Update(id, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, sponsor)
}

// Delete deletes a sponsor (admin).
func (h *SponsorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "sponsor deleted"})
}
