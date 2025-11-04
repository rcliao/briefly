package server

import (
	"briefly/internal/persistence"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// DigestListResponse represents the response for listing digests
type DigestListResponse struct {
	Digests []DigestSummary `json:"digests"`
	Total   int             `json:"total"`
}

// DigestSummary represents a summary of a digest for list views
type DigestSummary struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	ArticleCount  int       `json:"article_count"`
	ThemeCount    int       `json:"theme_count"`
	DateGenerated time.Time `json:"date_generated"`
}

// DigestDetailResponse represents the full digest response
type DigestDetailResponse struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	ExecutiveSummary string         `json:"executive_summary"`
	ArticleGroups    []ArticleGroup `json:"article_groups"`
	Metadata         DigestMetadata `json:"metadata"`
}

// ArticleGroup represents a theme with its articles
type ArticleGroup struct {
	Theme    string           `json:"theme"`
	Summary  string           `json:"summary"`
	Articles []ArticlePreview `json:"articles"`
}

// ArticlePreview represents an article in a digest
type ArticlePreview struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	URL                string   `json:"url"`
	Summary            string   `json:"summary"`
	ThemeRelevance     *float64 `json:"theme_relevance,omitempty"`
	ThemeRelevanceText string   `json:"theme_relevance_text,omitempty"`
}

// DigestMetadata represents digest generation metadata
type DigestMetadata struct {
	ArticleCount  int       `json:"article_count"`
	ThemeCount    int       `json:"theme_count"`
	DateGenerated time.Time `json:"date_generated"`
}

// handleListDigests handles GET /api/digests
func (s *Server) handleListDigests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// List digests from database
	digests, err := s.db.Digests().List(ctx, persistence.ListOptions{
		Limit:  50,  // Show last 50 digests
		Offset: 0,
	})

	if err != nil {
		s.log.Error("Failed to list digests", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to load digests")
		return
	}

	// Convert to response format
	summaries := make([]DigestSummary, len(digests))
	for i, digest := range digests {
		summaries[i] = DigestSummary{
			ID:            digest.ID,
			Title:         digest.Metadata.Title,
			ArticleCount:  digest.Metadata.ArticleCount,
			ThemeCount:    len(digest.ArticleGroups),
			DateGenerated: digest.Metadata.DateGenerated,
		}
	}

	s.respondJSON(w, http.StatusOK, DigestListResponse{
		Digests: summaries,
		Total:   len(summaries),
	})
}

// handleGetDigest handles GET /api/digests/{id}
func (s *Server) handleGetDigest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	digestID := chi.URLParam(r, "id")

	if digestID == "" {
		s.respondError(w, http.StatusBadRequest, "Digest ID is required")
		return
	}

	// Get digest from database
	digest, err := s.db.Digests().Get(ctx, digestID)
	if err != nil {
		s.log.Error("Failed to get digest", "id", digestID, "error", err)
		s.respondError(w, http.StatusNotFound, "Digest not found")
		return
	}

	// Build response with article groups
	articleGroups := make([]ArticleGroup, len(digest.ArticleGroups))
	for i, group := range digest.ArticleGroups {
		articles := make([]ArticlePreview, len(group.Articles))
		for j, article := range group.Articles {
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

			// Format relevance
			var relevanceText string
			if article.ThemeRelevanceScore != nil {
				relevanceText = formatRelevance(*article.ThemeRelevanceScore)
			}

			articles[j] = ArticlePreview{
				ID:                 article.ID,
				Title:              article.Title,
				URL:                article.URL,
				Summary:            summary,
				ThemeRelevance:     article.ThemeRelevanceScore,
				ThemeRelevanceText: relevanceText,
			}
		}

		articleGroups[i] = ArticleGroup{
			Theme:    group.Theme,
			Summary:  group.Summary,
			Articles: articles,
		}
	}

	response := DigestDetailResponse{
		ID:               digest.ID,
		Title:            digest.Metadata.Title,
		ExecutiveSummary: digest.DigestSummary,
		ArticleGroups:    articleGroups,
		Metadata: DigestMetadata{
			ArticleCount:  digest.Metadata.ArticleCount,
			ThemeCount:    len(digest.ArticleGroups),
			DateGenerated: digest.Metadata.DateGenerated,
		},
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleLatestDigest handles GET /api/digests/latest
func (s *Server) handleLatestDigest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the latest digest (first one in list)
	digests, err := s.db.Digests().List(ctx, persistence.ListOptions{
		Limit:  1,
		Offset: 0,
	})

	if err != nil {
		s.log.Error("Failed to get latest digest", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to load latest digest")
		return
	}

	if len(digests) == 0 {
		s.respondError(w, http.StatusNotFound, "No digests available")
		return
	}

	// Return the full digest detail (reuse the same logic)
	digest := digests[0]

	// Build response
	articleGroups := make([]ArticleGroup, len(digest.ArticleGroups))
	for i, group := range digest.ArticleGroups {
		articles := make([]ArticlePreview, len(group.Articles))
		for j, article := range group.Articles {
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

			// Format relevance
			var relevanceText string
			if article.ThemeRelevanceScore != nil {
				relevanceText = formatRelevance(*article.ThemeRelevanceScore)
			}

			articles[j] = ArticlePreview{
				ID:                 article.ID,
				Title:              article.Title,
				URL:                article.URL,
				Summary:            summary,
				ThemeRelevance:     article.ThemeRelevanceScore,
				ThemeRelevanceText: relevanceText,
			}
		}

		articleGroups[i] = ArticleGroup{
			Theme:    group.Theme,
			Summary:  group.Summary,
			Articles: articles,
		}
	}

	response := DigestDetailResponse{
		ID:               digest.ID,
		Title:            digest.Metadata.Title,
		ExecutiveSummary: digest.DigestSummary,
		ArticleGroups:    articleGroups,
		Metadata: DigestMetadata{
			ArticleCount:  digest.Metadata.ArticleCount,
			ThemeCount:    len(digest.ArticleGroups),
			DateGenerated: digest.Metadata.DateGenerated,
		},
	}

	s.respondJSON(w, http.StatusOK, response)
}

// formatRelevance formats a relevance score as a percentage string
func formatRelevance(score float64) string {
	percentage := int(score * 100)
	switch {
	case percentage >= 90:
		return "Highly Relevant (90%+)"
	case percentage >= 70:
		return "Very Relevant (70-89%)"
	case percentage >= 50:
		return "Relevant (50-69%)"
	default:
		return "Somewhat Relevant (<50%)"
	}
}

// respondError is defined in theme_handlers.go
