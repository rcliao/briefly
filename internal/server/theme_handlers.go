package server

import (
	"briefly/internal/core"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Theme API responses
type ThemeResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Enabled     bool     `json:"enabled"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type ThemeListResponse struct {
	Themes []ThemeResponse `json:"themes"`
	Total  int             `json:"total"`
}

type CreateThemeRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Enabled     bool     `json:"enabled"`
}

type UpdateThemeRequest struct {
	Description *string   `json:"description,omitempty"`
	Keywords    *[]string `json:"keywords,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
}

// handleListThemes handles GET /api/themes
func (s *Server) handleListThemes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check query parameter for filtering
	enabledOnly := r.URL.Query().Get("enabled") == "true"

	themes, err := s.db.Themes().List(ctx, enabledOnly)
	if err != nil {
		s.log.Error("Failed to list themes", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to retrieve themes")
		return
	}

	// Convert to response format
	themeResponses := make([]ThemeResponse, len(themes))
	for i, theme := range themes {
		themeResponses[i] = ThemeResponse{
			ID:          theme.ID,
			Name:        theme.Name,
			Description: theme.Description,
			Keywords:    theme.Keywords,
			Enabled:     theme.Enabled,
			CreatedAt:   theme.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   theme.UpdatedAt.Format(time.RFC3339),
		}
	}

	s.respondJSON(w, http.StatusOK, ThemeListResponse{
		Themes: themeResponses,
		Total:  len(themeResponses),
	})
}

// handleGetTheme handles GET /api/themes/{id}
func (s *Server) handleGetTheme(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	themeID := chi.URLParam(r, "id")

	theme, err := s.db.Themes().Get(ctx, themeID)
	if err != nil {
		s.log.Error("Failed to get theme", "id", themeID, "error", err)
		s.respondError(w, http.StatusNotFound, "Theme not found")
		return
	}

	s.respondJSON(w, http.StatusOK, ThemeResponse{
		ID:          theme.ID,
		Name:        theme.Name,
		Description: theme.Description,
		Keywords:    theme.Keywords,
		Enabled:     theme.Enabled,
		CreatedAt:   theme.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   theme.UpdatedAt.Format(time.RFC3339),
	})
}

// handleCreateTheme handles POST /api/themes
func (s *Server) handleCreateTheme(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "Theme name is required")
		return
	}

	// Check if theme already exists
	existing, _ := s.db.Themes().GetByName(ctx, req.Name)
	if existing != nil {
		s.respondError(w, http.StatusConflict, "Theme with this name already exists")
		return
	}

	// Create theme
	theme := &core.Theme{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Description: req.Description,
		Keywords:    req.Keywords,
		Enabled:     req.Enabled,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.db.Themes().Create(ctx, theme); err != nil {
		s.log.Error("Failed to create theme", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to create theme")
		return
	}

	s.respondJSON(w, http.StatusCreated, ThemeResponse{
		ID:          theme.ID,
		Name:        theme.Name,
		Description: theme.Description,
		Keywords:    theme.Keywords,
		Enabled:     theme.Enabled,
		CreatedAt:   theme.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   theme.UpdatedAt.Format(time.RFC3339),
	})
}

// handleUpdateTheme handles PATCH /api/themes/{id}
func (s *Server) handleUpdateTheme(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	themeID := chi.URLParam(r, "id")

	var req UpdateThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing theme
	theme, err := s.db.Themes().Get(ctx, themeID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Theme not found")
		return
	}

	// Apply updates
	if req.Description != nil {
		theme.Description = *req.Description
	}
	if req.Keywords != nil {
		theme.Keywords = *req.Keywords
	}
	if req.Enabled != nil {
		theme.Enabled = *req.Enabled
	}
	theme.UpdatedAt = time.Now().UTC()

	if err := s.db.Themes().Update(ctx, theme); err != nil {
		s.log.Error("Failed to update theme", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to update theme")
		return
	}

	s.respondJSON(w, http.StatusOK, ThemeResponse{
		ID:          theme.ID,
		Name:        theme.Name,
		Description: theme.Description,
		Keywords:    theme.Keywords,
		Enabled:     theme.Enabled,
		CreatedAt:   theme.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   theme.UpdatedAt.Format(time.RFC3339),
	})
}

// handleDeleteTheme handles DELETE /api/themes/{id}
func (s *Server) handleDeleteTheme(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	themeID := chi.URLParam(r, "id")

	if err := s.db.Themes().Delete(ctx, themeID); err != nil {
		s.log.Error("Failed to delete theme", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to delete theme")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondError writes an error response
func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"status":  status,
			"message": message,
		},
	})
}
