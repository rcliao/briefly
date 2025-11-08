package server

import (
	"briefly/internal/core"
	"briefly/internal/persistence"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Manual URL API responses
type ManualURLResponse struct {
	ID           string  `json:"id"`
	URL          string  `json:"url"`
	SubmittedBy  string  `json:"submitted_by"`
	Status       string  `json:"status"`
	ErrorMessage string  `json:"error_message,omitempty"`
	ProcessedAt  *string `json:"processed_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type ManualURLListResponse struct {
	URLs  []ManualURLResponse `json:"urls"`
	Total int                 `json:"total"`
}

type SubmitURLRequest struct {
	URLs        []string `json:"urls"`
	SubmittedBy string   `json:"submitted_by"`
}

type SubmitURLResponse struct {
	Submitted []ManualURLResponse `json:"submitted"`
	Failed    []struct {
		URL   string `json:"url"`
		Error string `json:"error"`
	} `json:"failed,omitempty"`
	Total int `json:"total"`
}

// handleListManualURLs handles GET /api/manual-urls
func (s *Server) handleListManualURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check query parameter for status filtering
	status := r.URL.Query().Get("status")

	var urls []core.ManualURL
	var err error

	if status != "" {
		// Filter by status
		urls, err = s.db.ManualURLs().GetByStatus(ctx, status, 100)
	} else {
		// Get all URLs
		urls, err = s.db.ManualURLs().List(ctx, persistence.ListOptions{Limit: 100})
	}

	if err != nil {
		s.log.Error("Failed to list manual URLs", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to retrieve URLs")
		return
	}

	// Convert to response format
	urlResponses := make([]ManualURLResponse, len(urls))
	for i, url := range urls {
		var processedAt *string
		if url.ProcessedAt != nil {
			formatted := url.ProcessedAt.Format(time.RFC3339)
			processedAt = &formatted
		}

		urlResponses[i] = ManualURLResponse{
			ID:           url.ID,
			URL:          url.URL,
			SubmittedBy:  url.SubmittedBy,
			Status:       url.Status,
			ErrorMessage: url.ErrorMessage,
			ProcessedAt:  processedAt,
			CreatedAt:    url.CreatedAt.Format(time.RFC3339),
		}
	}

	s.respondJSON(w, http.StatusOK, ManualURLListResponse{
		URLs:  urlResponses,
		Total: len(urlResponses),
	})
}

// handleGetManualURL handles GET /api/manual-urls/{id}
func (s *Server) handleGetManualURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlID := chi.URLParam(r, "id")

	url, err := s.db.ManualURLs().Get(ctx, urlID)
	if err != nil {
		s.log.Error("Failed to get manual URL", "id", urlID, "error", err)
		s.respondError(w, http.StatusNotFound, "URL not found")
		return
	}

	var processedAt *string
	if url.ProcessedAt != nil {
		formatted := url.ProcessedAt.Format(time.RFC3339)
		processedAt = &formatted
	}

	s.respondJSON(w, http.StatusOK, ManualURLResponse{
		ID:           url.ID,
		URL:          url.URL,
		SubmittedBy:  url.SubmittedBy,
		Status:       url.Status,
		ErrorMessage: url.ErrorMessage,
		ProcessedAt:  processedAt,
		CreatedAt:    url.CreatedAt.Format(time.RFC3339),
	})
}

// handleSubmitURLs handles POST /api/manual-urls
func (s *Server) handleSubmitURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SubmitURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if len(req.URLs) == 0 {
		s.respondError(w, http.StatusBadRequest, "At least one URL is required")
		return
	}

	response := SubmitURLResponse{
		Submitted: []ManualURLResponse{},
		Failed: []struct {
			URL   string `json:"url"`
			Error string `json:"error"`
		}{},
	}

	// Process each URL
	for _, urlStr := range req.URLs {
		// Check if URL already exists
		existing, _ := s.db.ManualURLs().GetByURL(ctx, urlStr)
		if existing != nil {
			response.Failed = append(response.Failed, struct {
				URL   string `json:"url"`
				Error string `json:"error"`
			}{
				URL:   urlStr,
				Error: "URL already exists",
			})
			continue
		}

		// Create manual URL entry
		manualURL := &core.ManualURL{
			ID:          uuid.NewString(),
			URL:         urlStr,
			SubmittedBy: req.SubmittedBy,
			Status:      core.ManualURLStatusPending,
			CreatedAt:   time.Now().UTC(),
		}

		if err := s.db.ManualURLs().Create(ctx, manualURL); err != nil {
			s.log.Error("Failed to create manual URL", "url", urlStr, "error", err)
			response.Failed = append(response.Failed, struct {
				URL   string `json:"url"`
				Error string `json:"error"`
			}{
				URL:   urlStr,
				Error: "Failed to store URL",
			})
			continue
		}

		response.Submitted = append(response.Submitted, ManualURLResponse{
			ID:          manualURL.ID,
			URL:         manualURL.URL,
			SubmittedBy: manualURL.SubmittedBy,
			Status:      manualURL.Status,
			CreatedAt:   manualURL.CreatedAt.Format(time.RFC3339),
		})
	}

	response.Total = len(response.Submitted)

	status := http.StatusCreated
	if len(response.Failed) > 0 && len(response.Submitted) == 0 {
		status = http.StatusBadRequest
	}

	s.respondJSON(w, status, response)
}

// handleRetryManualURL handles POST /api/manual-urls/{id}/retry
func (s *Server) handleRetryManualURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlID := chi.URLParam(r, "id")

	url, err := s.db.ManualURLs().Get(ctx, urlID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "URL not found")
		return
	}

	if url.Status != core.ManualURLStatusFailed {
		s.respondError(w, http.StatusBadRequest, "Only failed URLs can be retried")
		return
	}

	// Reset to pending
	if err := s.db.ManualURLs().UpdateStatus(ctx, urlID, string(core.ManualURLStatusPending), ""); err != nil {
		s.log.Error("Failed to retry URL", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to retry URL")
		return
	}

	// Fetch updated URL
	url, _ = s.db.ManualURLs().Get(ctx, urlID)

	s.respondJSON(w, http.StatusOK, ManualURLResponse{
		ID:          url.ID,
		URL:         url.URL,
		SubmittedBy: url.SubmittedBy,
		Status:      url.Status,
		CreatedAt:   url.CreatedAt.Format(time.RFC3339),
	})
}

// handleDeleteManualURL handles DELETE /api/manual-urls/{id}
func (s *Server) handleDeleteManualURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlID := chi.URLParam(r, "id")

	if err := s.db.ManualURLs().Delete(ctx, urlID); err != nil {
		s.log.Error("Failed to delete manual URL", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to delete URL")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
