package server

import (
	"briefly/internal/config"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server represents the HTTP server
type Server struct {
	router     *chi.Mux
	httpServer *http.Server
	db         persistence.Database
	config     config.Server
	log        *slog.Logger
	renderer   *TemplateRenderer
	analytics  interface{} // Optional analytics client
}

// New creates a new HTTP server instance
func New(db persistence.Database, cfg config.Server) *Server {
	log := logger.Get()

	// Initialize template renderer
	renderer, err := NewTemplateRenderer(true, "web/templates") // devMode=true for hot reload
	if err != nil {
		log.Warn("Failed to initialize template renderer, web pages may not work", "error", err)
	}

	s := &Server{
		router:    chi.NewRouter(),
		db:        db,
		config:    cfg,
		log:       log,
		renderer:  renderer,
		analytics: nil, // TODO: Initialize analytics client if configured
	}

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return s
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// Request ID middleware
	s.router.Use(middleware.RequestID)

	// Real IP middleware
	s.router.Use(middleware.RealIP)

	// Logging middleware
	s.router.Use(middleware.Logger)

	// Recovery middleware (recover from panics)
	s.router.Use(middleware.Recoverer)

	// Request timeout middleware
	s.router.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	if s.config.CORS.Enabled {
		s.router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.config.CORS.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300, // Maximum value not ignored by any major browsers
		}))
	}

	// Rate limiting middleware (simple throttle)
	if s.config.RateLimit.Enabled {
		// TODO: Implement proper rate limiting
		// For now, using basic throttle
		s.router.Use(middleware.Throttle(100))
	}
}

// setupRoutes configures routes for the server
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.Get("/health", s.handleHealth)

	// Status endpoint
	s.router.Get("/api/status", s.handleStatus)

	// API routes (will be extended in Phase 2)
	s.router.Route("/api", func(r chi.Router) {
		// Articles API
		r.Route("/articles", func(r chi.Router) {
			r.Get("/", s.handleListArticles)
			r.Get("/{id}", s.handleGetArticle)
			r.Get("/recent", s.handleRecentArticles)
		})

		// Digests API
		r.Route("/digests", func(r chi.Router) {
			r.Get("/", s.handleListDigests)
			r.Get("/{id}", s.handleGetDigest)
			r.Get("/latest", s.handleLatestDigest)
		})

		// Feeds API
		r.Route("/feeds", func(r chi.Router) {
			r.Get("/", s.handleListFeeds)
			r.Get("/{id}/stats", s.handleFeedStats)
		})

		// Themes API (Phase 0)
		r.Route("/themes", func(r chi.Router) {
			r.Get("/", s.handleListThemes)
			r.Post("/", s.handleCreateTheme)
			r.Get("/{id}", s.handleGetTheme)
			r.Patch("/{id}", s.handleUpdateTheme)
			r.Delete("/{id}", s.handleDeleteTheme)
		})

		// Manual URLs API (Phase 0)
		r.Route("/manual-urls", func(r chi.Router) {
			r.Get("/", s.handleListManualURLs)
			r.Post("/", s.handleSubmitURLs)
			r.Get("/{id}", s.handleGetManualURL)
			r.Post("/{id}/retry", s.handleRetryManualURL)
			r.Delete("/{id}", s.handleDeleteManualURL)
		})
	})

	// Web routes (HTML pages)
	s.router.Get("/", s.handleHomePage)
	s.router.Get("/digests/{id}", s.handleDigestDetailPage)
	s.router.Get("/themes", s.handleThemesPage)
	s.router.Get("/submit", s.handleSubmitPage)

	// HTMX partial routes
	s.router.Get("/api/digests/{id}/expand", s.handleExpandDigest)
	s.router.Get("/api/digests/{id}/collapse", s.handleCollapseDigest)

	// Static files (if directory exists)
	// TODO: Add static file serving in Phase 3
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.log.Info("Starting HTTP server",
		"addr", s.httpServer.Addr,
		"read_timeout", s.config.ReadTimeout,
		"write_timeout", s.config.WriteTimeout,
	)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down HTTP server gracefully...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log.Info("HTTP server stopped")
	return nil
}

// Router returns the chi router instance (useful for testing)
func (s *Server) Router() *chi.Mux {
	return s.router
}
