package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"godp/internal/api/dto"
	"godp/internal/config"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterInterceptor) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger returns an HTTP middleware that logs incoming requests with latency and status code.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		latency := time.Since(start)
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"latency_ms", latency.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// CORS handles Cross-Origin Resource Sharing and pre-flight OPTIONS requests.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Recovery catches panics, logs the error, and returns a 500 Internal Server Error JSON response.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("Panic recovered in HTTP handler", "panic", rec, "path", r.URL.Path)
				dto.Error(w, http.StatusInternalServerError, 500, "Internal server error occurred.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Security applies body size limits, security headers, and optional API key validation.
func Security(cfg *config.Config) func(http.Handler) http.Handler {
	maxBytes := cfg.MaxRequestSizeMB * 1024 * 1024

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Security Headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// 2. Body Size Limit
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}

			// 3. Optional API Key Authentication (if configured in environment)
			if cfg.APIKey != "" && !strings.HasPrefix(r.URL.Path, "/health") {
				key := r.Header.Get("X-API-Key")
				if key == "" {
					authHeader := r.Header.Get("Authorization")
					if strings.HasPrefix(authHeader, "Bearer ") {
						key = strings.TrimPrefix(authHeader, "Bearer ")
					}
				}

				if key != cfg.APIKey {
					dto.Error(w, http.StatusUnauthorized, 401, "Invalid or missing API key.")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
