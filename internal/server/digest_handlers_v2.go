package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// ArticleWithSummary extends Article with summary text for rendering
type ArticleWithSummary struct {
	ID                  string
	URL                 string
	Title               string
	ContentType         string
	CleanedText         string
	DatePublished       time.Time
	ThemeRelevanceScore *float64
	Summary             string
}

// handleExpandDigest returns the expanded digest content for HTMX
func (s *Server) handleExpandDigest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	digestID := chi.URLParam(r, "id")

	// Get the digest with all articles
	digest, err := s.db.Digests().Get(ctx, digestID)
	if err != nil {
		slog.Error("Failed to get digest", "error", err, "digest_id", digestID)
		http.Error(w, "Digest not found", http.StatusNotFound)
		return
	}

	// Enrich articles with summaries for rendering
	type EnrichedGroup struct {
		Theme    string
		Summary  string
		Articles []ArticleWithSummary
	}

	enrichedGroups := make([]EnrichedGroup, 0, len(digest.ArticleGroups))
	for _, group := range digest.ArticleGroups {
		enrichedArticles := make([]ArticleWithSummary, 0, len(group.Articles))
		for _, article := range group.Articles {
			// Find the summary for this article
			var summary string
			for _, s := range digest.Summaries {
				for _, aid := range s.ArticleIDs {
					if aid == article.ID {
						summary = s.SummaryText
						break
					}
				}
				if summary != "" {
					break
				}
			}

			enrichedArticles = append(enrichedArticles, ArticleWithSummary{
				ID:                  article.ID,
				URL:                 article.URL,
				Title:               article.Title,
				ContentType:         string(article.ContentType),
				CleanedText:         article.CleanedText,
				DatePublished:       article.DatePublished,
				ThemeRelevanceScore: article.ThemeRelevanceScore,
				Summary:             summary,
			})
		}

		enrichedGroups = append(enrichedGroups, EnrichedGroup{
			Theme:    group.Theme,
			Summary:  group.Summary,
			Articles: enrichedArticles,
		})
	}

	// Create view model
	viewData := struct {
		ID            string
		DigestSummary string
		ArticleGroups []EnrichedGroup
	}{
		ID:            digest.ID,
		DigestSummary: digest.DigestSummary,
		ArticleGroups: enrichedGroups,
	}

	// Render the expanded digest partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "partials/digest-expanded.html", viewData); err != nil {
		slog.Error("Failed to render expanded digest", "error", err)
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		return
	}

	// Track expansion event (TODO: implement analytics tracking)
	// if s.analytics != nil {
	// 	s.analytics.TrackEvent(ctx, "digest_expanded", map[string]interface{}{
	// 		"digest_id": digestID,
	// 	})
	// }
}

// handleCollapseDigest returns empty content to collapse the digest
func (s *Server) handleCollapseDigest(w http.ResponseWriter, r *http.Request) {
	// ctx := r.Context()
	// digestID := chi.URLParam(r, "id")

	// Return empty HTML to clear the container
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte("")); err != nil {
		slog.Error("Failed to write response", "error", err)
	}

	// Track collapse event (TODO: implement analytics tracking)
	// if s.analytics != nil {
	// 	s.analytics.TrackEvent(ctx, "digest_collapsed", map[string]interface{}{
	// 		"digest_id": digestID,
	// 	})
	// }
}
