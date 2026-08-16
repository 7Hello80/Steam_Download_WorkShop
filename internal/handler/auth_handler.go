package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"steam-download-tool/internal/model"
	"steam-download-tool/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Register(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Login(&req)
	if err != nil {
		status := http.StatusUnauthorized
		if strings.Contains(err.Error(), "邮箱未验证") {
			status = http.StatusForbidden
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	clientID := h.svc.GetGitHubClientID()
	if clientID == "" {
		writeError(w, http.StatusBadRequest, "GitHub OAuth is not configured")
		return
	}

	redirectURL := "https://github.com/login/oauth/authorize" +
		"?client_id=" + clientID +
		"&redirect_uri=" + h.svc.GetGitHubRedirectURL() +
		"&scope=user:email"

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.VerifyEmail(req.Email, req.Code)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) ResendCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.ResendCode(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "验证码已重新发送"})
}

func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	resp, err := h.svc.GitHubLogin(code)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	frontendURL := h.svc.GetFrontendURL()
	http.Redirect(w, r, frontendURL+"/auth/github/callback?token="+resp.Token, http.StatusFound)
}
