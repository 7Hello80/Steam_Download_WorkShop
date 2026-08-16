package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"steam-download-tool/internal/config"
	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db       *sql.DB
	cfg      *config.Config
	emailSvc *EmailService
	httpClient *http.Client
}

func NewAuthService(db *sql.DB, cfg *config.Config, emailSvc *EmailService) *AuthService {
	return &AuthService{
		db:         db,
		cfg:        cfg,
		emailSvc:   emailSvc,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *AuthService) Register(req *model.RegisterRequest) (*model.RegisterResponse, error) {
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return nil, errors.New("email, username, and password are required")
	}
	if !isValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}
	if len(req.Username) < 2 || len(req.Username) > 32 {
		return nil, errors.New("username must be between 2 and 32 characters")
	}

	// Check if email already exists in users OR pending registrations
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", req.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, errors.New("email already registered")
	}
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM pending_registrations WHERE email = ?)", req.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check pending: %w", err)
	}
	if exists {
		// If there's an existing pending registration, delete it first (allow re-registration)
		_, _ = s.db.Exec("DELETE FROM pending_registrations WHERE email = ?", req.Email)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Generate verification code
	code := generateVerificationCode()
	codeExpiresAt := time.Now().Add(5 * time.Minute)

	// Store in pending_registrations — account is NOT created yet,
	// only after email verification will the user record be inserted into users table.
	_, err = s.db.Exec(
		"INSERT INTO pending_registrations (id, email, username, password_hash, verification_code, code_expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		uuid.New().String(), req.Email, req.Username, string(hash), code, codeExpiresAt, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("insert pending registration: %w", err)
	}

	// Send verification email
	if err := s.emailSvc.SendVerificationEmail(req.Email, code); err != nil {
		// Log error but don't fail — user can request resend
		fmt.Printf("WARNING: failed to send verification email to %s: %v\n", req.Email, err)
	}

	return &model.RegisterResponse{
		Message: "验证码已发送至您的邮箱，请查收并完成验证后即可登录",
		Email:   req.Email,
	}, nil
}

// VerifyEmail validates the verification code, creates the actual user account,
// and returns an auth token. The account is only created after successful verification.
// Also supports legacy unverified accounts already in the users table.
func (s *AuthService) VerifyEmail(email, code string) (*model.AuthResponse, error) {
	if email == "" || code == "" {
		return nil, errors.New("email and verification code are required")
	}

	// Check if already a verified user
	var alreadyVerified bool
	s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ? AND email_verified = 1)", email).Scan(&alreadyVerified)
	if alreadyVerified {
		return nil, errors.New("该邮箱已通过验证，请直接登录")
	}

	// First, try pending_registrations (new flow)
	var pendingID, storedCode, username, passwordHash string
	var codeExpiresAt time.Time

	err := s.db.QueryRow(
		"SELECT id, username, password_hash, verification_code, code_expires_at FROM pending_registrations WHERE email = ?",
		email,
	).Scan(&pendingID, &username, &passwordHash, &storedCode, &codeExpiresAt)

	if err == nil {
		// Found in pending_registrations — new flow
		if storedCode != code {
			return nil, errors.New("验证码错误")
		}
		if time.Now().After(codeExpiresAt) {
			return nil, errors.New("验证码已过期，请重新发送")
		}

		// Create the actual user account
		userID := uuid.New().String()
		now := time.Now()

		_, err = s.db.Exec(
			"INSERT INTO users (id, email, username, password_hash, role, email_verified, created_at, updated_at) VALUES (?, ?, ?, ?, 'user', 1, ?, ?)",
			userID, email, username, passwordHash, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}

		// Delete from pending_registrations
		_, _ = s.db.Exec("DELETE FROM pending_registrations WHERE id = ?", pendingID)

		user := &model.User{
			ID:            userID,
			Email:         email,
			Username:      username,
			PasswordHash:  passwordHash,
			Role:          "user",
			EmailVerified: true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		token, err := s.generateToken(user)
		if err != nil {
			return nil, err
		}

		return &model.AuthResponse{Token: token, User: user}, nil
	}

	// Not found in pending_registrations — try legacy users table (backward compatibility)
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("query pending registration: %w", err)
	}

	var legacyUserID string
	var legacyStoredCode sql.NullString
	var legacyCodeExpiresAt sql.NullTime

	err = s.db.QueryRow(
		"SELECT id, verification_code, verification_code_expires_at FROM users WHERE email = ? AND email_verified = 0",
		email,
	).Scan(&legacyUserID, &legacyStoredCode, &legacyCodeExpiresAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("该邮箱未注册或已验证，请重新注册")
	}
	if err != nil {
		return nil, fmt.Errorf("query legacy user: %w", err)
	}

	if !legacyStoredCode.Valid || legacyStoredCode.String == "" {
		return nil, errors.New("验证码不存在，请重新发送")
	}
	if legacyStoredCode.String != code {
		return nil, errors.New("验证码错误")
	}
	if !legacyCodeExpiresAt.Valid || time.Now().After(legacyCodeExpiresAt.Time) {
		return nil, errors.New("验证码已过期，请重新发送")
	}

	// Mark legacy user as email verified
	_, err = s.db.Exec(
		"UPDATE users SET email_verified = 1, verification_code = NULL, verification_code_expires_at = NULL, updated_at = ? WHERE id = ?",
		time.Now(), legacyUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("update legacy user: %w", err)
	}

	// Fetch the full user
	fullUser, err := s.GetProfile(legacyUserID)
	if err != nil {
		return nil, err
	}
	fullUser.EmailVerified = true

	token, err := s.generateToken(fullUser)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{Token: token, User: fullUser}, nil
}

// ResendCode generates a new verification code and sends it to the email.
func (s *AuthService) ResendCode(email string) error {
	if email == "" {
		return errors.New("email is required")
	}

	// Check if already verified
	var alreadyVerified bool
	s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ? AND email_verified = 1)", email).Scan(&alreadyVerified)
	if alreadyVerified {
		return errors.New("该邮箱已通过验证，请直接登录")
	}

	code := generateVerificationCode()
	codeExpiresAt := time.Now().Add(5 * time.Minute)

	// Try pending_registrations first (new flow)
	var pendingExists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM pending_registrations WHERE email = ?)", email).Scan(&pendingExists)
	if err != nil {
		return fmt.Errorf("check pending: %w", err)
	}

	if pendingExists {
		_, err = s.db.Exec(
			"UPDATE pending_registrations SET verification_code = ?, code_expires_at = ? WHERE email = ?",
			code, codeExpiresAt, email,
		)
		if err != nil {
			return fmt.Errorf("update pending verification code: %w", err)
		}
	} else {
		// Legacy flow: check users table for unverified account
		var legacyExists bool
		err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ? AND email_verified = 0)", email).Scan(&legacyExists)
		if err != nil {
			return fmt.Errorf("check legacy user: %w", err)
		}
		if !legacyExists {
			return errors.New("该邮箱未注册，请先注册")
		}

		_, err = s.db.Exec(
			"UPDATE users SET verification_code = ?, verification_code_expires_at = ?, updated_at = ? WHERE email = ? AND email_verified = 0",
			code, codeExpiresAt, time.Now(), email,
		)
		if err != nil {
			return fmt.Errorf("update legacy verification code: %w", err)
		}
	}

	if err := s.emailSvc.SendVerificationEmail(email, code); err != nil {
		return fmt.Errorf("发送验证邮件失败: %w", err)
	}

	return nil
}

func (s *AuthService) Login(req *model.LoginRequest) (*model.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	user := &model.User{}
	err := s.db.QueryRow(
		"SELECT id, email, username, COALESCE(password_hash,''), COALESCE(github_id,''), COALESCE(avatar_url,''), role, email_verified, created_at, updated_at FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.GitHubID, &user.AvatarURL, &user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("invalid email or password")
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !user.EmailVerified {
		return nil, errors.New("邮箱未验证，请先验证邮箱后再登录")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) GitHubLogin(code string) (*model.AuthResponse, error) {
	if s.cfg.GitHubClientID == "" || s.cfg.GitHubClientSecret == "" {
		return nil, errors.New("GitHub OAuth is not configured (set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET)")
	}

	// Step 1: Exchange code for access token
	accessToken, err := s.exchangeGitHubCode(code)
	if err != nil {
		return nil, fmt.Errorf("exchange GitHub code: %w", err)
	}

	// Step 2: Fetch user info from GitHub API
	githubUser, err := s.fetchGitHubUser(accessToken)
	if err != nil {
		return nil, fmt.Errorf("fetch GitHub user: %w", err)
	}

	// Step 3: Fetch user emails if primary email not set
	email := githubUser["email"]
	if email == "" {
		emails, err := s.fetchGitHubEmails(accessToken)
		if err != nil {
			return nil, fmt.Errorf("fetch GitHub emails: %w", err)
		}
		email = emails // use primary email
	}
	if email == "" {
		return nil, errors.New("could not retrieve email from GitHub - make sure your GitHub account has a public email")
	}

	username := githubUser["login"]
	if username == "" {
		username = githubUser["name"]
	}
	if username == "" {
		username = "github_user"
	}

	avatarURL := githubUser["avatar_url"]
	githubID := fmt.Sprintf("%v", githubUser["id"])

	return s.GitHubLoginByUser(githubID, username, email, avatarURL)
}

// exchangeGitHubCode exchanges an OAuth code for an access token.
func (s *AuthService) exchangeGitHubCode(code string) (string, error) {
	reqBody := url.Values{
		"client_id":     {s.cfg.GitHubClientID},
		"client_secret": {s.cfg.GitHubClientSecret},
		"code":          {code},
		"redirect_uri":  {s.cfg.GitHubRedirectURL},
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(reqBody.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if errMsg, ok := result["error"]; ok {
		return "", fmt.Errorf("GitHub OAuth error: %v - %v", errMsg, result["error_description"])
	}

	token, ok := result["access_token"].(string)
	if !ok || token == "" {
		return "", errors.New("no access token in GitHub response")
	}

	return token, nil
}

// fetchGitHubUser fetches the authenticated user's info from GitHub API.
func (s *AuthService) fetchGitHubUser(accessToken string) (map[string]string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode user response: %w", err)
	}

	user := make(map[string]string)
	for _, key := range []string{"id", "login", "name", "email", "avatar_url"} {
		if v, ok := result[key]; ok && v != nil {
			user[key] = fmt.Sprintf("%v", v)
		}
	}
	return user, nil
}

// fetchGitHubEmails fetches the user's emails from GitHub API and returns the primary one.
func (s *AuthService) fetchGitHubEmails(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("emails request failed: %w", err)
	}
	defer resp.Body.Close()

	var emails []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode emails response: %w", err)
	}

	// Find primary email, or fall back to the first verified one
	for _, email := range emails {
		if primary, ok := email["primary"].(bool); ok && primary {
			if addr, ok := email["email"].(string); ok {
				return addr, nil
			}
		}
	}
	for _, email := range emails {
		if verified, ok := email["verified"].(bool); ok && verified {
			if addr, ok := email["email"].(string); ok {
				return addr, nil
			}
		}
	}

	return "", nil
}

func (s *AuthService) GitHubLoginByUser(githubUserID, username, email, avatarURL string) (*model.AuthResponse, error) {
	// Find existing user by github_id
	user := &model.User{}
	err := s.db.QueryRow(
		"SELECT id, email, username, COALESCE(password_hash,''), COALESCE(github_id,''), COALESCE(avatar_url,''), role, email_verified, created_at, updated_at FROM users WHERE github_id = ?",
		githubUserID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.GitHubID, &user.AvatarURL, &user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		// Create new user (GitHub users are auto-verified)
		user = &model.User{
			ID:            uuid.New().String(),
			Email:         email,
			Username:      username,
			GitHubID:      githubUserID,
			AvatarURL:     avatarURL,
			Role:          "user",
			EmailVerified: true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		_, err = s.db.Exec(
			"INSERT INTO users (id, email, username, github_id, avatar_url, role, email_verified, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)",
			user.ID, user.Email, user.Username, user.GitHubID, user.AvatarURL, user.Role, user.CreatedAt, user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert github user: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("query github user: %w", err)
	} else {
		// Update avatar URL if changed
		if user.AvatarURL != avatarURL {
			_, _ = s.db.Exec("UPDATE users SET avatar_url = ?, updated_at = ? WHERE id = ?", avatarURL, time.Now(), user.ID)
			user.AvatarURL = avatarURL
		}
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) GetProfile(userID string) (*model.User, error) {
	user := &model.User{}
	err := s.db.QueryRow(
		"SELECT id, email, username, COALESCE(password_hash,''), COALESCE(github_id,''), COALESCE(avatar_url,''), role, email_verified, created_at, updated_at FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.GitHubID, &user.AvatarURL, &user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

func (s *AuthService) UpdateProfile(userID string, req *model.UpdateProfileRequest) (*model.User, error) {
	if req.Username == "" {
		return nil, errors.New("username is required")
	}
	if len(req.Username) < 2 || len(req.Username) > 32 {
		return nil, errors.New("username must be between 2 and 32 characters")
	}

	if req.AvatarURL != "" {
		_, err := s.db.Exec("UPDATE users SET username = ?, avatar_url = ?, updated_at = ? WHERE id = ?",
			req.Username, req.AvatarURL, time.Now(), userID)
		if err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	} else {
		_, err := s.db.Exec("UPDATE users SET username = ?, updated_at = ? WHERE id = ?",
			req.Username, time.Now(), userID)
		if err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	}

	return s.GetProfile(userID)
}

// UpdateAvatar updates the user's avatar URL in the database.
func (s *AuthService) UpdateAvatar(userID string, avatarURL string) (*model.User, error) {
	if avatarURL == "" {
		return nil, errors.New("avatar URL is required")
	}

	_, err := s.db.Exec("UPDATE users SET avatar_url = ?, updated_at = ? WHERE id = ?",
		avatarURL, time.Now(), userID)
	if err != nil {
		return nil, fmt.Errorf("update avatar: %w", err)
	}

	return s.GetProfile(userID)
}

func (s *AuthService) GetGitHubClientID() string   { return s.cfg.GitHubClientID }
func (s *AuthService) GetGitHubRedirectURL() string { return s.cfg.GitHubRedirectURL }
func (s *AuthService) GetFrontendURL() string       { return s.cfg.FrontendURL }

func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := &middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "steam-download-tool",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// isValidEmail performs basic email format validation.
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	// Must have a dot in the domain
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

// generateVerificationCode generates a random 6-digit verification code.
func generateVerificationCode() string {
	code := rand.Intn(1000000)
	return fmt.Sprintf("%06d", code)
}
