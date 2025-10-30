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
	Version string            `json:"version"`
	Uptime  string            `json:"uptime"`
	Database DatabaseStatus   `json:"database"`
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

// handleListDigests handles GET /api/digests
func (s *Server) handleListDigests(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Digests API - Coming in Phase 2",
		"data":    []interface{}{},
	})
}

// handleGetDigest handles GET /api/digests/{id}
func (s *Server) handleGetDigest(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Single digest API - Coming in Phase 2",
	})
}

// handleLatestDigest handles GET /api/digests/latest
func (s *Server) handleLatestDigest(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 2
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Latest digest API - Coming in Phase 2",
	})
}

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

// handleHomePage handles GET / (HTML page)
func (s *Server) handleHomePage(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement in Phase 3
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Briefly - News Aggregator</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2/dist/tailwind.min.css" rel="stylesheet">
</head>
<body class="bg-gray-50">
    <div class="container mx-auto px-4 py-16">
        <div class="max-w-2xl mx-auto text-center">
            <h1 class="text-5xl font-bold text-blue-600 mb-4">Briefly</h1>
            <p class="text-xl text-gray-700 mb-8">LLM-Powered News Aggregator</p>

            <div class="bg-white rounded-lg shadow-lg p-8 mb-8">
                <h2 class="text-2xl font-semibold mb-4">Phase 1 Complete ✅</h2>
                <p class="text-gray-600 mb-4">HTTP server is up and running!</p>

                <div class="space-y-2 text-left">
                    <div class="flex items-center">
                        <span class="text-green-500 mr-2">✓</span>
                        <span>Health check endpoint</span>
                    </div>
                    <div class="flex items-center">
                        <span class="text-green-500 mr-2">✓</span>
                        <span>Status API</span>
                    </div>
                    <div class="flex items-center">
                        <span class="text-green-500 mr-2">✓</span>
                        <span>Graceful shutdown</span>
                    </div>
                    <div class="flex items-center">
                        <span class="text-green-500 mr-2">✓</span>
                        <span>CORS support</span>
                    </div>
                </div>
            </div>

            <div class="bg-blue-50 rounded-lg p-6 mb-8">
                <h3 class="text-lg font-semibold mb-3">Try These Endpoints:</h3>
                <div class="space-y-2 text-sm">
                    <div class="bg-white rounded px-4 py-2">
                        <code class="text-blue-600">GET /health</code> - Health check
                    </div>
                    <div class="bg-white rounded px-4 py-2">
                        <code class="text-blue-600">GET /api/status</code> - Server status
                    </div>
                    <div class="bg-white rounded px-4 py-2">
                        <code class="text-blue-600">GET /api/articles</code> - Articles API (Phase 2)
                    </div>
                </div>
            </div>

            <div class="text-gray-500 text-sm">
                <p>Phase 2: REST API Endpoints - Coming soon</p>
                <p>Phase 3: Web Frontend - Coming soon</p>
            </div>
        </div>
    </div>
</body>
</html>
	`)); err != nil {
		s.log.Error("Failed to write response", "error", err)
	}
}

// respondJSON writes a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.log.Error("Failed to encode JSON response", "error", err)
	}
}

// respondError writes an error response (to be used in Phase 2)
// Commented out to avoid unused function warning
// func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
// 	s.respondJSON(w, status, map[string]interface{}{
// 		"error": map[string]interface{}{
// 			"status":  status,
// 			"message": message,
// 		},
// 	})
// }
