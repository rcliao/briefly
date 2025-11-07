package server

import (
	"briefly/internal/core"
	"briefly/internal/persistence"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"
)

// HomePageData contains all data needed for the homepage
type HomePageData struct {
	Themes           []ThemeWithCount
	Digests          []DigestSummaryView
	ActiveTheme      string
	AllCount         int
	TotalDigests     int
	TotalArticles    int
	TotalThemes      int
	LatestDigestDate time.Time

	// Pagination
	HasMore          bool
	HasPrevious      bool
	CurrentPage      int
	TotalPages       int
	NextPage         int
	PreviousPage     int
	PageSize         int

	CurrentYear      int

	// PostHog integration
	PostHogEnabled bool
	PostHogAPIKey  string
	PostHogHost    string
}

// ThemeWithCount includes digest count for each theme
type ThemeWithCount struct {
	ID          string
	Name        string
	Description string
	Keywords    []string
	Enabled     bool
	DigestCount int
}

// DigestSummaryView is the view model for digest cards
type DigestSummaryView struct {
	ID             string
	Themes         []string
	DigestSummary  string
	SummaryPreview string // Truncated summary for homepage preview (~100 chars)
	Metadata       DigestMetadataView
}

// DigestMetadataView contains digest metadata for the view
type DigestMetadataView struct {
	Title        string
	ArticleCount int
	ThemeCount   int
	DateGenerated time.Time
	QualityScore float64
}

// handleHomePage renders the homepage with theme tabs and digests
func (s *Server) handleHomePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get theme filter from query param
	themeID := r.URL.Query().Get("theme")
	if themeID == "all" {
		themeID = ""
	}

	// Get page parameter (default to 1)
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := parsePageNumber(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Check if this is an HTMX request for partial update
	if isHTMXRequest(r) {
		s.handleHomePagePartial(w, r, ctx, themeID, page)
		return
	}

	// Full page render
	data, err := s.getHomePageData(ctx, themeID, page)
	if err != nil {
		slog.Error("Failed to get homepage data", "error", err)
		http.Error(w, "Failed to load homepage", http.StatusInternalServerError)
		return
	}

	// Render the home page (it will automatically use base layout via block inheritance)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "pages/home.html", data); err != nil {
		slog.Error("Failed to render homepage", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}

	// Track page view (TODO: implement analytics tracking)
	// if s.analytics != nil {
	// 	s.analytics.TrackEvent(ctx, "homepage_viewed", map[string]interface{}{
	// 		"theme":        themeID,
	// 		"digest_count": len(data.Digests),
	// 	})
	// }
}

// handleHomePagePartial renders just the digest list and pagination for HTMX requests
func (s *Server) handleHomePagePartial(w http.ResponseWriter, r *http.Request, ctx context.Context, themeID string, page int) {
	// Get full page data with pagination
	data, err := s.getHomePageData(ctx, themeID, page)
	if err != nil {
		slog.Error("Failed to get homepage data", "error", err, "theme", themeID, "page", page)
		http.Error(w, "Failed to load digests", http.StatusInternalServerError)
		return
	}

	// Render digest list and pagination together
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "partials/digest-list-with-pagination.html", data); err != nil {
		slog.Error("Failed to render digest list", "error", err)
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		return
	}
}

const digestsPerPage = 10

// getHomePageData retrieves all data needed for the homepage
func (s *Server) getHomePageData(ctx context.Context, activeThemeID string, page int) (*HomePageData, error) {
	// Get all enabled themes
	themes, err := s.db.Themes().ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	// Filter to enabled themes only and convert to view model
	themesWithCount := []ThemeWithCount{}
	for _, theme := range themes {
		if theme.Enabled {
			// Get digest count for this theme (TODO: implement in repository)
			count := 0 // Placeholder
			themesWithCount = append(themesWithCount, ThemeWithCount{
				ID:          theme.ID,
				Name:        theme.Name,
				Description: theme.Description,
				Keywords:    theme.Keywords,
				Enabled:     theme.Enabled,
				DigestCount: count,
			})
		}
	}

	// Get total count of digests for pagination calculation
	var totalDigests int
	if activeThemeID == "" {
		// Count all digests
		allDigests, err := s.db.Digests().List(ctx, persistence.ListOptions{Limit: 10000, Offset: 0})
		if err != nil {
			slog.Warn("Failed to get digest count", "error", err)
		}
		totalDigests = len(allDigests)
	} else {
		// Count digests for specific theme (TODO: optimize with dedicated count query)
		allDigests, err := s.db.Digests().List(ctx, persistence.ListOptions{Limit: 10000, Offset: 0})
		if err != nil {
			slog.Warn("Failed to get digest count", "error", err)
		}
		theme, err := s.db.Themes().Get(ctx, activeThemeID)
		if err != nil {
			slog.Warn("Failed to lookup theme", "theme_id", activeThemeID, "error", err)
			theme = &core.Theme{Name: activeThemeID}
		}
		themeName := theme.Name

		// Count digests containing this theme
		for _, d := range allDigests {
			for _, group := range d.ArticleGroups {
				if group.Theme == themeName {
					totalDigests++
					break
				}
			}
		}
	}

	// Calculate pagination
	totalPages := (totalDigests + digestsPerPage - 1) / digestsPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Get digests for current page (filtered by theme if specified)
	digests, err := s.getDigestsForTheme(ctx, activeThemeID, page)
	if err != nil {
		return nil, err
	}

	// Get stats for overview
	allDigests, err := s.db.Digests().List(ctx, persistence.ListOptions{Limit: 1000, Offset: 0})
	if err != nil {
		slog.Warn("Failed to get digest stats", "error", err)
	}

	var latestDate time.Time
	if len(allDigests) > 0 {
		latestDate = allDigests[0].Metadata.DateGenerated
	}

	// Count total articles (approximate from digests)
	totalArticles := 0
	for _, d := range allDigests {
		totalArticles += d.Metadata.ArticleCount
	}

	// PostHog configuration (TODO: add to config struct)
	postHogEnabled := false
	postHogAPIKey := ""
	postHogHost := "https://app.posthog.com"

	return &HomePageData{
		Themes:           themesWithCount,
		Digests:          digests,
		ActiveTheme:      activeThemeID,
		AllCount:         len(allDigests),
		TotalDigests:     totalDigests,
		TotalArticles:    totalArticles,
		TotalThemes:      len(themesWithCount),
		LatestDigestDate: latestDate,

		// Pagination
		HasMore:          page < totalPages,
		HasPrevious:      page > 1,
		CurrentPage:      page,
		TotalPages:       totalPages,
		NextPage:         page + 1,
		PreviousPage:     page - 1,
		PageSize:         digestsPerPage,

		CurrentYear:      time.Now().Year(),
		PostHogEnabled:   postHogEnabled,
		PostHogAPIKey:    postHogAPIKey,
		PostHogHost:      postHogHost,
	}, nil
}

// getDigestsForTheme retrieves digests, optionally filtered by theme, with pagination
func (s *Server) getDigestsForTheme(ctx context.Context, themeID string, page int) ([]DigestSummaryView, error) {
	var digests []core.Digest
	var err error

	// Calculate offset from page number
	offset := (page - 1) * digestsPerPage

	if themeID == "" {
		// Get paginated digests (all themes)
		digests, err = s.db.Digests().List(ctx, persistence.ListOptions{
			Limit:  digestsPerPage,
			Offset: offset,
		})
	} else {
		// Look up the theme name from the ID
		// ArticleGroups store theme names, not IDs
		theme, err := s.db.Themes().Get(ctx, themeID)
		if err != nil {
			slog.Warn("Failed to lookup theme by ID", "theme_id", themeID, "error", err)
			// If lookup fails, fall back to using the ID as-is (maybe it's already a name)
			theme = &core.Theme{Name: themeID}
		}

		themeName := theme.Name

		// Get digests for specific theme
		// For now, get all and filter in memory (TODO: optimize with database query)
		allDigests, err := s.db.Digests().List(ctx, persistence.ListOptions{Limit: 10000, Offset: 0})
		if err != nil {
			return nil, err
		}

		// Filter digests that contain articles with this theme
		var filteredDigests []core.Digest
		for _, d := range allDigests {
			hasTheme := false
			for _, group := range d.ArticleGroups {
				if group.Theme == themeName {
					hasTheme = true
					break
				}
			}
			if hasTheme {
				filteredDigests = append(filteredDigests, d)
			}
		}

		// Apply pagination to filtered results
		start := offset
		end := offset + digestsPerPage
		if start > len(filteredDigests) {
			start = len(filteredDigests)
		}
		if end > len(filteredDigests) {
			end = len(filteredDigests)
		}
		digests = filteredDigests[start:end]
	}

	if err != nil {
		return nil, err
	}

	// Convert to view models
	result := make([]DigestSummaryView, 0, len(digests))
	for _, d := range digests {
		// Extract unique themes
		themeSet := make(map[string]bool)
		for _, group := range d.ArticleGroups {
			themeSet[group.Theme] = true
		}
		themes := make([]string, 0, len(themeSet))
		for theme := range themeSet {
			themes = append(themes, theme)
		}
		// Sort themes alphabetically for consistent display order
		sort.Strings(themes)

		result = append(result, DigestSummaryView{
			ID:             d.ID,
			Themes:         themes,
			DigestSummary:  d.DigestSummary,
			SummaryPreview: d.Metadata.TLDRSummary, // Use generated TL;DR summary
			Metadata: DigestMetadataView{
				Title:         d.Metadata.Title,
				ArticleCount:  d.Metadata.ArticleCount,
				ThemeCount:    len(d.ArticleGroups), // Calculate from article groups
				DateGenerated: d.Metadata.DateGenerated,
				QualityScore:  d.Metadata.QualityScore,
			},
		})
	}

	return result, nil
}

// parsePageNumber parses a page number from a string
func parsePageNumber(s string) (int, error) {
	var page int
	_, err := fmt.Sscanf(s, "%d", &page)
	return page, err
}
