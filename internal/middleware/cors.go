package middleware

import (
	"net/http"
	"strings"
)

// CORS returns a middleware that restricts cross-origin requests to the
// configured frontend URL. If frontendURL is empty, it falls back to "*".
func CORS(frontendURL string) func(http.Handler) http.Handler {
	allowOrigin := frontendURL
	if allowOrigin == "" {
		allowOrigin = "*"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set basic CORS headers always for preflight and simple requests
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges, Content-Disposition")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Vary", "Origin")

			if allowOrigin == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				// Allow if origin matches the configured frontend URL
				configuredOrigin := strings.TrimRight(allowOrigin, "/")
				if origin == configuredOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			} else {
				// Same-origin request (no Origin header) — set the configured origin
				// to satisfy crossorigin="anonymous" on same-origin <script> tags
				w.Header().Set("Access-Control-Allow-Origin", strings.TrimRight(allowOrigin, "/"))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
