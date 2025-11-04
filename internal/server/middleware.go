package server

import (
	"log/slog"
	"net/http"
	"os"
)

// requireAdminAPI middleware protects admin endpoints with an API key
// The API key should be set in the ADMIN_API_KEY environment variable
func (s *Server) requireAdminAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the admin API key from environment
		adminAPIKey := os.Getenv("ADMIN_API_KEY")

		// If no API key is set, block all admin requests
		if adminAPIKey == "" {
			slog.Warn("Admin API accessed but ADMIN_API_KEY not set")
			http.Error(w, "Admin API is disabled. Set ADMIN_API_KEY environment variable to enable.", http.StatusForbidden)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <api_key>"
		expectedAuth := "Bearer " + adminAPIKey
		if authHeader != expectedAuth {
			slog.Warn("Invalid admin API key attempt", "remote_addr", r.RemoteAddr)
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// API key is valid, proceed
		next.ServeHTTP(w, r)
	})
}

// securityHeaders adds security headers to all responses
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (basic)
		// Allow inline scripts/styles for HTMX and daisyUI
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://unpkg.com https://cdn.tailwindcss.com; " +
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' https://app.posthog.com;"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}

// mobileOptimized adds headers to improve mobile experience
func mobileOptimized(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Enable mobile viewport scaling
		w.Header().Set("X-UA-Compatible", "IE=edge")

		next.ServeHTTP(w, r)
	})
}

// noCache adds headers to prevent caching (useful for HTMX partials)
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		next.ServeHTTP(w, r)
	})
}

// cacheStaticAssets adds caching headers for static assets
func cacheStaticAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cache static assets for 1 year
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		next.ServeHTTP(w, r)
	})
}
