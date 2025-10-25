package persistence

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"briefly/internal/logger"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// MigrationManager handles database migrations
type MigrationManager struct {
	db  *PostgresDB
	log *slog.Logger
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *PostgresDB) *MigrationManager {
	return &MigrationManager{
		db:  db,
		log: logger.Get(),
	}
}

// Migrate runs all pending migrations
func (m *MigrationManager) Migrate(ctx context.Context) error {
	m.log.Info("Starting database migration")

	// Create schema_migrations table if it doesn't exist
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get all available migrations
	available, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Find pending migrations
	pending := m.findPendingMigrations(available, applied)

	if len(pending) == 0 {
		m.log.Info("No pending migrations")
		return nil
	}

	m.log.Info("Found pending migrations", "count", len(pending))

	// Apply each pending migration
	for _, migration := range pending {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	m.log.Info("Migration completed successfully", "applied", len(pending))
	return nil
}

// Status shows migration status
func (m *MigrationManager) Status(ctx context.Context) ([]MigrationStatus, error) {
	// Ensure migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	available, err := m.loadMigrations()
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[int]bool)
	for _, v := range applied {
		appliedMap[v] = true
	}

	var status []MigrationStatus
	for _, migration := range available {
		status = append(status, MigrationStatus{
			Version:     migration.Version,
			Description: migration.Description,
			Applied:     appliedMap[migration.Version],
		})
	}

	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     int
	Description string
	Applied     bool
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist
func (m *MigrationManager) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`
	_, err := m.db.db.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations returns a list of applied migration versions
func (m *MigrationManager) getAppliedMigrations(ctx context.Context) ([]int, error) {
	query := `SELECT version FROM schema_migrations ORDER BY version`
	rows, err := m.db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, rows.Err()
}

// loadMigrations loads all migration files from the embedded filesystem
func (m *MigrationManager) loadMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Parse version from filename (e.g., "001_initial_schema.sql" -> 1)
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			m.log.Warn("Skipping migration file with invalid format", "file", entry.Name())
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			m.log.Warn("Skipping migration file with invalid version", "file", entry.Name())
			continue
		}

		// Extract description from filename
		description := strings.TrimSuffix(parts[1], ".sql")
		description = strings.ReplaceAll(description, "_", " ")

		// Read migration SQL
		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version:     version,
			Description: description,
			SQL:         string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// findPendingMigrations returns migrations that haven't been applied yet
func (m *MigrationManager) findPendingMigrations(available []Migration, applied []int) []Migration {
	appliedMap := make(map[int]bool)
	for _, version := range applied {
		appliedMap[version] = true
	}

	var pending []Migration
	for _, migration := range available {
		if !appliedMap[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending
}

// applyMigration applies a single migration in a transaction
func (m *MigrationManager) applyMigration(ctx context.Context, migration Migration) error {
	m.log.Info("Applying migration", "version", migration.Version, "description", migration.Description)

	tx, err := m.db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Note: The migration SQL already inserts into schema_migrations,
	// but we need to ensure it's recorded even if the migration doesn't include it
	// This is a safeguard for future migrations that might not include the INSERT
	_, err = tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (version, description)
		VALUES ($1, $2)
		ON CONFLICT (version) DO NOTHING
	`, migration.Version, migration.Description)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	m.log.Info("Successfully applied migration", "version", migration.Version)
	return nil
}

// Rollback rolls back the last migration (use with caution!)
func (m *MigrationManager) Rollback(ctx context.Context) error {
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	lastVersion := applied[len(applied)-1]
	m.log.Warn("Rolling back migration", "version", lastVersion)

	// Delete the migration record
	_, err = m.db.db.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, lastVersion)
	if err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	m.log.Info("Migration rolled back - you must manually revert database changes", "version", lastVersion)
	return nil
}
