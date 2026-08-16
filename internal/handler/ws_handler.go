package handler

import (
	"net/http"

	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/ws"
)

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) Upgrade(w http.ResponseWriter, r *http.Request) {
	// JWT token passed as query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}

	// Validate JWT token (reuse middleware logic)
	// For simplicity, we validate manually here
	claims, err := middleware.ValidateToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := h.hub.HandleUpgrade(w, r, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "websocket upgrade failed")
	}
}
