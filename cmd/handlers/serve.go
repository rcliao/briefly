package handlers

import (
	"briefly/internal/config"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"briefly/internal/server"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// NewServeCmd creates the serve command for starting the HTTP server
func NewServeCmd() *cobra.Command {
	var (
		port        int
		host        string
		staticDir   string
		templateDir string
		reload      bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server for web interface",
		Long: `Start the briefly web server to browse aggregated articles.

The server provides:
  • Web UI for browsing articles and digests
  • REST API for programmatic access
  • Health check and status endpoints

The server reads from the database populated by 'briefly aggregate'.
Run aggregation separately (e.g., via cron) to keep content fresh.

Examples:
  # Start server on default port 8080
  briefly serve

  # Start on custom port
  briefly serve --port 3000

  # Start with custom directories
  briefly serve --static-dir ./static --template-dir ./templates

  # Start with auto-reload (development mode)
  briefly serve --reload`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), port, host, staticDir, templateDir, reload)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "HTTP server port (default from config: 8080)")
	cmd.Flags().StringVar(&host, "host", "", "HTTP server host (default from config: 0.0.0.0)")
	cmd.Flags().StringVar(&staticDir, "static-dir", "", "Static files directory (default from config)")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Template directory (default from config)")
	cmd.Flags().BoolVar(&reload, "reload", false, "Auto-reload templates in dev mode (not yet implemented)")

	return cmd
}

func runServe(ctx context.Context, port int, host, staticDir, templateDir string, reload bool) error {
	log := logger.Get()
	log.Info("Starting HTTP server")

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override server config from flags if provided
	serverCfg := cfg.Server
	if port != 0 {
		serverCfg.Port = port
	}
	if host != "" {
		serverCfg.Host = host
	}
	if staticDir != "" {
		serverCfg.StaticDir = staticDir
	}
	if templateDir != "" {
		serverCfg.TemplateDir = templateDir
	}

	// Get database connection string
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		// Try environment variable fallback
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return fmt.Errorf("database connection string not configured\n\n" +
				"The web server requires a database connection. Please set one of:\n" +
				"  • database.connection_string in .briefly.yaml\n" +
				"  • DATABASE_URL environment variable\n\n" +
				"Example:\n" +
				"  export DATABASE_URL='postgres://user:pass@localhost:5432/briefly?sslmode=disable'\n")
		}
	}

	// Connect to database
	log.Info("Connecting to database")
	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w\n\n"+
			"Make sure PostgreSQL is running and the connection string is correct.\n"+
			"Run 'briefly migrate up' to initialize the database schema.", err)
	}

	log.Info("Database connection successful")

	// Create HTTP server
	srv := server.New(db, serverCfg)

	// Channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Info(fmt.Sprintf("Server listening on http://%s:%d", serverCfg.Host, serverCfg.Port))
		log.Info("Press Ctrl+C to stop")
		serverErrors <- srv.Start()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive our signal or an error from server
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info("Server shutdown initiated", "signal", sig.String())

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverCfg.ShutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(shutdownCtx); err != nil {
			// Force close if graceful shutdown fails
			log.Error("Server shutdown failed, forcing close", "error", err)
			return fmt.Errorf("server shutdown failed: %w", err)
		}

		log.Info("Server stopped successfully")
	}

	return nil
}
