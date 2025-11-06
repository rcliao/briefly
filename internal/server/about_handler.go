package server

import (
	"net/http"
)

// handleAboutPage renders the static about page
func (s *Server) handleAboutPage(w http.ResponseWriter, r *http.Request) {
	// Minimal data needed for base template (PostHog analytics)
	data := map[string]interface{}{
		"PostHogEnabled": s.analytics != nil,
		"PostHogAPIKey":  "",
		"PostHogHost":    "https://app.posthog.com",
	}

	// TODO: Get PostHog config from server if needed
	// For now, PostHog is initialized in base.html template

	// Render the about page (it will automatically use base layout via block inheritance)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "pages/about.html", data); err != nil {
		s.log.Error("Failed to render about page", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}
