package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"steam-download-tool/internal/config"
	"steam-download-tool/internal/crypto"
	"steam-download-tool/internal/database"
	"steam-download-tool/internal/handler"
	"steam-download-tool/internal/middleware"
	"steam-download-tool/internal/queue"
	"steam-download-tool/internal/service"
	"steam-download-tool/internal/storage"
	"steam-download-tool/internal/ws"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

//go:embed web/dist
var frontendFS embed.FS

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize AES encryption key
	if cfg.AESKey == "" {
		log.Fatalf("AES encryption key is required. Set aes_key in config.yaml or AES_KEY environment variable.")
	}
	if err := crypto.InitKey(cfg.AESKey); err != nil {
		log.Fatalf("Failed to initialize crypto: %v", err)
	}

	// Initialize JWT secret
	middleware.SetJWTSecret(cfg.JWTSecret)

	// Ensure runtime directories
	if err := storage.EnsureDir(cfg.OutputDir); err != nil {
		log.Fatalf("Failed to create output dir: %v", err)
	}
	if err := storage.EnsureDir(cfg.StaticDir); err != nil {
		log.Fatalf("Failed to create static dir: %v", err)
	}
	if err := storage.EnsureDir("./file/avatar"); err != nil {
		log.Fatalf("Failed to create file/avatar dir: %v", err)
	}

	// Validate DepotDownloader binary
	if _, err := os.Stat(cfg.DepotDownloaderPath); os.IsNotExist(err) {
		log.Printf("WARNING: DepotDownloader not found at %s - downloads will fail", cfg.DepotDownloaderPath)
	}

	// Open database
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// Initialize services
	emailSvc := service.NewEmailService(cfg)
	authSvc := service.NewAuthService(db, cfg, emailSvc)
	userSvc := service.NewUserService(db)
	adminSvc := service.NewAdminService(db)
	dlSvc := service.NewDownloadService(db)
	announcementSvc := service.NewAnnouncementService(db)
	sponsorSvc := service.NewSponsorService(db)

	// Initialize WebSocket hub
	wsHub := ws.NewHub()

	// Initialize queue
	q := queue.NewQueue(db, wsHub, cfg)
	defer q.Stop()

	// Initialize handlers
	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(authSvc, userSvc)
	adminH := handler.NewAdminHandler(adminSvc)
	dlH := handler.NewDownloadHandler(dlSvc, userSvc, q, cfg)
	fileH := handler.NewFileHandler(dlSvc, cfg)
	convertH := handler.NewConvertHandler(dlSvc, cfg)
	wsH := handler.NewWSHandler(wsHub)
	announcementH := handler.NewAnnouncementHandler(announcementSvc)
	sponsorH := handler.NewSponsorHandler(sponsorSvc)

	// Build router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.FrontendURL))
	r.Use(chimiddleware.RealIP)

	// API routes with body size limit (1MB)
	r.Group(func(r chi.Router) {
		r.Use(middleware.MaxBodySize(1 << 20))

		// Public routes
		r.Route("/api/auth", func(r chi.Router) {
			r.Post("/register", authH.Register)
			r.Post("/login", authH.Login)
				r.Post("/verify-email", authH.VerifyEmail)
			r.Post("/resend-code", authH.ResendCode)
			r.Get("/github", authH.GitHubLogin)
			r.Get("/github/callback", authH.GitHubCallback)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(cfg.JWTSecret))

			// User profile
			r.Get("/api/user/profile", userH.GetProfile)
			r.Put("/api/user/profile", userH.UpdateProfile)
			r.Post("/api/user/avatar", userH.UploadAvatar)

			// Steam credentials
			r.Post("/api/steam/credentials", userH.SaveSteamCredentials)
			r.Get("/api/steam/credentials", userH.GetSteamCredentials)
			r.Delete("/api/steam/credentials/{id}", userH.DeleteSteamCredentials)

			// Downloads
			r.Post("/api/downloads", dlH.Start)
			r.Get("/api/downloads", dlH.List)
			r.Get("/api/downloads/{id}", dlH.Get)
			r.Get("/api/downloads/{id}/output", dlH.GetOutput)
			r.Post("/api/downloads/{id}/cancel", dlH.Cancel)

			// Files
			r.Get("/api/files", fileH.List)
			r.Delete("/api/files/{id}", fileH.Delete)
			r.Get("/api/files/{id}/download", fileH.Download)

			// Workshop conversion (zip → mpkg)
			r.Get("/api/convert/list", convertH.ListConvertible)
			r.Post("/api/convert", convertH.Convert)
			r.Get("/api/convert/{id}/download", convertH.Download)

			// Queue info
			r.Get("/api/queue", func(w http.ResponseWriter, r *http.Request) {
				info := q.GetQueueInfo(middleware.GetUserID(r))
				handler.WriteJSON(w, http.StatusOK, info)
			})

			// Announcements (active, for all logged-in users)
			r.Get("/api/announcements", announcementH.ListActive)

			// Sponsors (visible, for all logged-in users)
			r.Get("/api/sponsors", sponsorH.ListVisible)

			// WebSocket
			r.Get("/api/ws", wsH.Upgrade)

			// Admin routes (requires admin role)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)

				r.Get("/api/admin/dashboard", adminH.GetDashboard)
				r.Get("/api/admin/users", adminH.ListUsers)
				r.Get("/api/admin/users/{id}", adminH.GetUser)
				r.Put("/api/admin/users/{id}/role", adminH.UpdateUserRole)
				r.Post("/api/admin/users/{id}/ban", adminH.BanUser)
				r.Post("/api/admin/users/{id}/unban", adminH.UnbanUser)
				r.Delete("/api/admin/users/{id}", adminH.DeleteUser)

				// Announcement management
				r.Get("/api/admin/announcements", announcementH.ListAll)
				r.Post("/api/admin/announcements", announcementH.Create)
				r.Put("/api/admin/announcements/{id}", announcementH.Update)
				r.Delete("/api/admin/announcements/{id}", announcementH.Delete)

				// Sponsor management
				r.Get("/api/admin/sponsors", sponsorH.ListAll)
				r.Post("/api/admin/sponsors", sponsorH.Create)
				r.Put("/api/admin/sponsors/{id}", sponsorH.Update)
				r.Delete("/api/admin/sponsors/{id}", sponsorH.Delete)
			})
		})
	})

	// Serve static files (completed downloads) — public direct download, no auth
	staticFS := http.FileServer(http.Dir(cfg.StaticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", staticFS))

	// Serve avatar files — public
	avatarFS := http.FileServer(http.Dir("./file/avatar"))
	r.Handle("/file/avatar/*", http.StripPrefix("/file/avatar/", avatarFS))

	// Serve embedded frontend SPA
	serveFrontend(r)

	// Start cleanup scheduler
	go startCleanupScheduler(db, cfg)

	// Start HTTP server with tuned settings for large file downloads
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("Payment mode: %s", cfg.PaymentMode)
	log.Printf("Max workers: %d", cfg.MaxWorkers)
	log.Printf("File TTL: %d hours", cfg.FileTTLHours)

	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   0, // Disabled — large file downloads may take hours
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		q.Stop()
		srv.Close()
		db.Close()
		os.Exit(0)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// serveFrontend serves the embedded Vue SPA with fallback to index.html.
func serveFrontend(r chi.Router) {
	distFS, err := fs.Sub(frontendFS, "web/dist")
	if err != nil {
		log.Printf("WARNING: Frontend not embedded (web/dist/ not found). Run build.sh first.")
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/static/") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(`<!DOCTYPE html><html><head><title>Steam Download Tool</title></head>
<body><h1>Frontend not built</h1><p>Run <code>./build.sh</code> to build the frontend.</p></body></html>`))
		})
		return
	}

	// Read index.html once at startup for SPA fallback
	indexHTML, indexErr := fs.ReadFile(distFS, "index.html")
	if indexErr != nil {
		log.Printf("WARNING: Failed to read embedded index.html: %v", indexErr)
	}

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Skip API and static routes
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/static/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Normalize path
		urlPath := r.URL.Path
		if urlPath == "/" {
			urlPath = "/index.html"
		}

		// Try to read and serve the requested file directly from embedded FS
		filePath := strings.TrimPrefix(urlPath, "/")
		data, readErr := fs.ReadFile(distFS, filePath)
		if readErr != nil {
			// File not found — serve index.html for SPA client-side routing
			if indexHTML == nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHTML)
			return
		}

		// Set correct Content-Type based on file extension
		ext := filepath.Ext(filePath)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.Write(data)
	})
}

// startCleanupScheduler periodically deletes expired files.
func startCleanupScheduler(db *sql.DB, cfg *config.Config) {
	// Run immediately on startup
	runCleanup(db, cfg)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		runCleanup(db, cfg)
	}
}

func runCleanup(db *sql.DB, cfg *config.Config) {
	log.Printf("Running cleanup for expired files and stale registrations")

	// Clean up expired pending registrations (older than 24 hours)
	if _, err := db.Exec("DELETE FROM pending_registrations WHERE code_expires_at < datetime('now', '-24 hours')"); err != nil {
		log.Printf("Cleanup: failed to clean pending registrations: %v", err)
	}

	// Query for tasks that are completed and past their expires_at
	rows, err := db.Query(
		`SELECT id FROM download_tasks WHERE status = ? AND expires_at IS NOT NULL AND expires_at < datetime('now')`,
		"completed",
	)
	if err != nil {
		log.Printf("Cleanup: failed to query expired tasks: %v", err)
		return
	}
	defer rows.Close()

	var expiredIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		expiredIDs = append(expiredIDs, id)
	}

	for _, taskID := range expiredIDs {
		taskDir := filepath.Join(cfg.StaticDir, taskID)

		// Remove the file directory
		if err := os.RemoveAll(taskDir); err != nil {
			log.Printf("Cleanup: failed to remove %s: %v", taskDir, err)
		} else {
			log.Printf("Cleanup: removed expired task directory %s", taskDir)
		}

		// Delete task record entirely — no longer keep expired entries
		if _, err := db.Exec("DELETE FROM download_tasks WHERE id = ?", taskID); err != nil {
			log.Printf("Cleanup: failed to delete task %s record: %v", taskID, err)
		} else {
			log.Printf("Cleanup: deleted expired task %s record", taskID)
		}
	}

	// Also clean up any orphaned directories in static/ that have no corresponding DB task
	entries, err := os.ReadDir(cfg.StaticDir)
	if err != nil {
		log.Printf("Cleanup: failed to read static dir: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()

		// Check if this directory corresponds to a task in the DB
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM download_tasks WHERE id = ?)", dirName).Scan(&exists)
		if err != nil || !exists {
			// Orphaned directory, remove it
			orphanDir := filepath.Join(cfg.StaticDir, dirName)
			if err := os.RemoveAll(orphanDir); err != nil {
				log.Printf("Cleanup: failed to remove orphan directory %s: %v", orphanDir, err)
			} else {
				log.Printf("Cleanup: removed orphan directory %s", orphanDir)
			}
		}
	}
}
