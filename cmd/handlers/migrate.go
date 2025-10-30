package handlers

import (
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewMigrateCmd creates the migrate command for database migrations
func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage database migrations",
		Long: `Manage database schema migrations.

Subcommands:
  up       Apply all pending migrations
  status   Show migration status
  rollback Roll back the last migration (use with caution!)

The migration system tracks applied migrations in the schema_migrations table
and applies new migrations in sequential order.

Examples:
  # Apply all pending migrations
  briefly migrate up

  # Check migration status
  briefly migrate status

  # Rollback last migration (manual data cleanup required)
  briefly migrate rollback`,
	}

	cmd.AddCommand(newMigrateUpCmd())
	cmd.AddCommand(newMigrateStatusCmd())
	cmd.AddCommand(newMigrateRollbackCmd())

	return cmd
}

func newMigrateUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		Long: `Apply all pending database migrations.

This command will:
  • Create schema_migrations table if it doesn't exist
  • Check which migrations have been applied
  • Apply all pending migrations in order
  • Record each migration in schema_migrations

Migrations are applied in a transaction and will rollback on failure.

Example:
  briefly migrate up`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateUp(cmd.Context())
		},
	}
}

func newMigrateStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long: `Show the status of all migrations.

Displays which migrations have been applied and which are pending.

Example:
  briefly migrate status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateStatus(cmd.Context())
		},
	}
}

func newMigrateRollbackCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Roll back the last migration",
		Long: `Roll back the last applied migration.

⚠️  WARNING: This only removes the migration record from schema_migrations.
    You must manually revert any database schema changes!

This is a dangerous operation and should only be used in development.
Use --force to skip confirmation prompt.

Example:
  briefly migrate rollback
  briefly migrate rollback --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateRollback(cmd.Context(), force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

// Implementation functions

func runMigrateUp(ctx context.Context) error {
	log := logger.Get()
	log.Info("Starting database migration")

	// Get database connection
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Create migration manager
	pgDB, ok := db.(*persistence.PostgresDB)
	if !ok {
		return fmt.Errorf("only PostgreSQL database is supported for migrations")
	}

	migrator := persistence.NewMigrationManager(pgDB)

	// Run migrations
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("✅ All migrations applied successfully")
	return nil
}

func runMigrateStatus(ctx context.Context) error {
	// Get database connection
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Create migration manager
	pgDB, ok := db.(*persistence.PostgresDB)
	if !ok {
		return fmt.Errorf("only PostgreSQL database is supported for migrations")
	}

	migrator := persistence.NewMigrationManager(pgDB)

	// Get status
	status, err := migrator.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	if len(status) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	fmt.Println("📊 Migration Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-10s %-10s %s\n", "Version", "Status", "Description")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	appliedCount := 0
	pendingCount := 0

	for _, m := range status {
		statusStr := "pending"
		statusIcon := "⏳"
		if m.Applied {
			statusStr = "applied"
			statusIcon = "✅"
			appliedCount++
		} else {
			pendingCount++
		}

		fmt.Printf("%-10d %s %-8s %s\n", m.Version, statusIcon, statusStr, m.Description)
	}

	fmt.Println()
	fmt.Printf("Applied: %d | Pending: %d | Total: %d\n", appliedCount, pendingCount, len(status))

	if pendingCount > 0 {
		fmt.Println("\nRun 'briefly migrate up' to apply pending migrations")
	}

	return nil
}

func runMigrateRollback(ctx context.Context, force bool) error {
	log := logger.Get()

	if !force {
		fmt.Println("⚠️  WARNING: Rolling back migrations is dangerous!")
		fmt.Println("This will only remove the migration record from schema_migrations.")
		fmt.Println("You must manually revert any database schema changes.")
		fmt.Println()
		fmt.Print("Are you sure you want to proceed? (yes/no): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if response != "yes" {
			fmt.Println("Rollback cancelled")
			return nil
		}
	}

	// Get database connection
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Create migration manager
	pgDB, ok := db.(*persistence.PostgresDB)
	if !ok {
		return fmt.Errorf("only PostgreSQL database is supported for migrations")
	}

	migrator := persistence.NewMigrationManager(pgDB)

	// Rollback
	if err := migrator.Rollback(ctx); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	log.Warn("Migration record removed - remember to manually revert database changes")
	fmt.Println("⚠️  Migration record removed")
	fmt.Println("You must manually revert database changes")

	return nil
}
