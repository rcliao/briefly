package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/rcliao/briefly/internal/core"
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
	HasMore          bool
	NextOffset       int
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
	ID            string
	Themes        []string
	DigestSummary string
	Metadata      DigestMetadataView
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

	// Check if this is an HTMX request for partial update
	if isHTMXRequest(r) {
		s.handleHomePagePartial(w, r, ctx, themeID)
		return
	}

	// Full page render
	data, err := s.getHomePageData(ctx, themeID)
	if err != nil {
		slog.Error("Failed to get homepage data", "error", err)
		http.Error(w, "Failed to load homepage", http.StatusInternalServerError)
		return
	}

	// Render the full page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "pages/home.html", data); err != nil {
		slog.Error("Failed to render homepage", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}

	// Track page view
	if s.analytics != nil {
		s.analytics.TrackEvent(ctx, "homepage_viewed", map[string]interface{}{
			"theme":        themeID,
			"digest_count": len(data.Digests),
		})
	}
}

// handleHomePagePartial renders just the digest list for HTMX requests
func (s *Server) handleHomePagePartial(w http.ResponseWriter, r *http.Request, ctx context.Context, themeID string) {
	// Get digests for the theme
	digests, err := s.getDigestsForTheme(ctx, themeID)
	if err != nil {
		slog.Error("Failed to get digests", "error", err, "theme", themeID)
		http.Error(w, "Failed to load digests", http.StatusInternalServerError)
		return
	}

	// Render only the digest-list partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Render(w, "partials/digest-list.html", digests); err != nil {
		slog.Error("Failed to render digest list", "error", err)
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		return
	}
}

// getHomePageData retrieves all data needed for the homepage
func (s *Server) getHomePageData(ctx context.Context, activeThemeID string) (*HomePageData, error) {
	// Get all enabled themes
	themes, err := s.repos.Theme.List(ctx)
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

	// Get digests (filtered by theme if specified)
	digests, err := s.getDigestsForTheme(ctx, activeThemeID)
	if err != nil {
		return nil, err
	}

	// Get stats
	allDigests, err := s.repos.Digest.List(ctx, 1000, 0)
	if err != nil {
		slog.Warn("Failed to get digest count", "error", err)
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

	// PostHog configuration
	postHogEnabled := s.config.GetString("posthog.api_key") != ""
	postHogAPIKey := s.config.GetString("posthog.api_key")
	postHogHost := s.config.GetString("posthog.host")
	if postHogHost == "" {
		postHogHost = "https://app.posthog.com"
	}

	return &HomePageData{
		Themes:           themesWithCount,
		Digests:          digests,
		ActiveTheme:      activeThemeID,
		AllCount:         len(allDigests),
		TotalDigests:     len(allDigests),
		TotalArticles:    totalArticles,
		TotalThemes:      len(themesWithCount),
		LatestDigestDate: latestDate,
		HasMore:          false, // TODO: implement pagination
		NextOffset:       0,
		CurrentYear:      time.Now().Year(),
		PostHogEnabled:   postHogEnabled,
		PostHogAPIKey:    postHogAPIKey,
		PostHogHost:      postHogHost,
	}, nil
}

// getDigestsForTheme retrieves digests, optionally filtered by theme
func (s *Server) getDigestsForTheme(ctx context.Context, themeID string) ([]DigestSummaryView, error) {
	var digests []core.Digest
	var err error

	if themeID == "" {
		// Get all digests
		digests, err = s.repos.Digest.List(ctx, 20, 0)
	} else {
		// Get digests for specific theme (TODO: implement in repository)
		// For now, get all and filter in memory
		allDigests, err := s.repos.Digest.List(ctx, 100, 0)
		if err != nil {
			return nil, err
		}

		// Filter digests that contain articles with this theme
		for _, d := range allDigests {
			hasTheme := false
			for _, group := range d.ArticleGroups {
				if group.Theme == themeID || containsThemeByID(group, themeID) {
					hasTheme = true
					break
				}
			}
			if hasTheme {
				digests = append(digests, d)
			}
		}
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

		result = append(result, DigestSummaryView{
			ID:            d.ID,
			Themes:        themes,
			DigestSummary: d.DigestSummary,
			Metadata: DigestMetadataView{
				Title:         d.Metadata.Title,
				ArticleCount:  d.Metadata.ArticleCount,
				ThemeCount:    d.Metadata.ThemeCount,
				DateGenerated: d.Metadata.DateGenerated,
				QualityScore:  d.Metadata.QualityScore,
			},
		})
	}

	return result, nil
}

// containsThemeByID checks if an article group contains a theme with the given ID
func containsThemeByID(group core.ArticleGroup, themeID string) bool {
	// This is a placeholder - you may need to implement theme ID lookup
	// For now, we're matching by theme name
	return false
}
