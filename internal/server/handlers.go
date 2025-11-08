package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// Health check response
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// Status response
type StatusResponse struct {
	Version  string         `json:"version"`
	Uptime   string         `json:"uptime"`
	Database DatabaseStatus `json:"database"`
}

// DatabaseStatus represents database health
type DatabaseStatus struct {
	Connected bool `json:"connected"`
	Articles  int  `json:"articles"`
	Digests   int  `json:"digests"`
	Feeds     int  `json:"feeds"`
}

var serverStartTime = time.Now()

// handleHealth handles the /health endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)

	// Check database connection
	if err := s.db.Ping(r.Context()); err != nil {
		checks["database"] = "error"
		s.respondJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status: "unhealthy",
			Checks: checks,
		})
		return
	}

	checks["database"] = "ok"

	s.respondJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
		Checks: checks,
	})
}

// handleStatus handles the /api/status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(serverStartTime)

	// Get database statistics
	dbStatus := DatabaseStatus{
		Connected: true,
	}

	// TODO: Query actual counts from database
	// For now, just mark as connected

	s.respondJSON(w, http.StatusOK, StatusResponse{
		Version:  "v3.2.0-dev",
		Uptime:   uptime.String(),
		Database: dbStatus,
	})
}

// handleListArticles handles GET /api/articles
func (s *Server) handleListArticles(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Articles API - Coming in Phase 2",
		"data":    []interface{}{},
	})
}

// handleGetArticle handles GET /api/articles/{id}
func (s *Server) handleGetArticle(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Single article API - Coming in Phase 2",
	})
}

// handleRecentArticles handles GET /api/articles/recent
func (s *Server) handleRecentArticles(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Recent articles API - Coming in Phase 2",
		"data":    []interface{}{},
	})
}

// Digest handlers are now in digest_handlers.go

// handleListFeeds handles GET /api/feeds
func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Feeds API - Coming in Phase 2",
		"data":    []interface{}{},
	})
}

// handleFeedStats handles GET /api/feeds/{id}/stats
func (s *Server) handleFeedStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Feed stats API - Coming in Phase 2",
	})
}

// respondJSON writes a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.log.Error("Failed to encode JSON response", "error", err)
	}
}

// respondError is now in digest_handlers.go
