package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleExpandDigest returns the expanded digest content for HTMX
func (s *Server) handleExpandDigest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	digestID := chi.URLParam(r, "id")

	// Get the digest with all articles
	digest, err := s.repos.Digest.GetByID(ctx, digestID)
	if err != nil {
		slog.Error("Failed to get digest", "error", err, "digest_id", digestID)
		http.Error(w, "Digest not found", http.StatusNotFound)
		return
	}

	// Render the expanded digest partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "partials/digest-expanded.html", digest); err != nil {
		slog.Error("Failed to render expanded digest", "error", err)
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		return
	}

	// Track expansion event
	if s.analytics != nil {
		s.analytics.TrackEvent(ctx, "digest_expanded", map[string]interface{}{
			"digest_id": digestID,
		})
	}
}

// handleCollapseDigest returns empty content to collapse the digest
func (s *Server) handleCollapseDigest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	digestID := chi.URLParam(r, "id")

	// Return empty HTML to clear the container
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(""))

	// Track collapse event
	if s.analytics != nil {
		s.analytics.TrackEvent(ctx, "digest_collapsed", map[string]interface{}{
			"digest_id": digestID,
		})
	}
}
