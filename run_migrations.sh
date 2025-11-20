#!/bin/bash

# Run all pending database migrations in order
# This script applies migrations sequentially and handles errors gracefully

set -e  # Exit on error

echo "🗄️  Database Migration Runner"
echo "============================"
echo ""

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo "❌ No .env file found"
    exit 1
fi

# Verify DATABASE_URL
if [ -z "$DATABASE_URL" ]; then
    echo "❌ DATABASE_URL not set in .env"
    exit 1
fi

echo "✅ Connected to database"
echo ""

# Ensure schema_migrations table exists
echo "📋 Ensuring schema_migrations table exists..."
psql "$DATABASE_URL" -c "
    CREATE TABLE IF NOT EXISTS schema_migrations (
        version INT PRIMARY KEY,
        description TEXT NOT NULL,
        applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
    );
" > /dev/null 2>&1
echo "   ✅ Schema migrations table ready"
echo ""

# Get list of applied migrations
echo "🔍 Checking applied migrations..."
APPLIED_MIGRATIONS=$(psql "$DATABASE_URL" -tAc "SELECT version FROM schema_migrations ORDER BY version;")
echo "   Applied migrations: $(echo "$APPLIED_MIGRATIONS" | wc -l | xargs)"
echo ""

# Function to check if migration is applied
is_migration_applied() {
    local version=$1
    echo "$APPLIED_MIGRATIONS" | grep -q "^${version}$"
}

# Function to apply a single migration
apply_migration() {
    local file=$1
    local version=$(echo "$file" | grep -oE '^[0-9]+')
    local description=$(basename "$file" .sql | sed 's/^[0-9]*_//')

    if is_migration_applied "$version"; then
        echo "   ⏭️  Migration $version already applied: $description"
        return 0
    fi

    echo "   🔄 Applying migration $version: $description"

    # Apply the migration
    if psql "$DATABASE_URL" -f "$file" > /tmp/migration_$version.log 2>&1; then
        echo "   ✅ Migration $version completed"

        # Show any notices (helpful for pgvector installation messages)
        if grep -q "NOTICE" /tmp/migration_$version.log; then
            echo ""
            grep "NOTICE" /tmp/migration_$version.log | sed 's/^NOTICE:  /      /'
            echo ""
        fi
    else
        echo "   ❌ Migration $version failed"
        echo ""
        echo "Error log:"
        cat /tmp/migration_$version.log
        echo ""
        return 1
    fi
}

# Apply migrations in order
echo "🚀 Applying pending migrations..."
echo ""

MIGRATION_COUNT=0
MIGRATION_DIR="internal/persistence/migrations"

# Find all migration files and sort them numerically
for migration_file in $(ls "$MIGRATION_DIR"/*.sql | sort -V); do
    if apply_migration "$migration_file"; then
        ((MIGRATION_COUNT++)) || true
    else
        echo ""
        echo "❌ Migration failed. Stopping."
        exit 1
    fi
done

echo ""
echo "✅ All migrations completed!"
echo ""

# Show final status
echo "📊 Database Status:"
psql "$DATABASE_URL" -c "
    SELECT
        version,
        description,
        applied_at
    FROM schema_migrations
    ORDER BY version
    LIMIT 10;
"

echo ""
echo "🔍 Extensions Enabled:"
psql "$DATABASE_URL" -c "
    SELECT
        extname as extension,
        extversion as version
    FROM pg_extension
    WHERE extname IN ('vector', 'uuid-ossp', 'pg_trgm')
    ORDER BY extname;
"

echo ""
echo "📈 Table Sizes:"
psql "$DATABASE_URL" -c "
    SELECT
        schemaname || '.' || tablename as table,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
    FROM pg_tables
    WHERE schemaname = 'public'
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
    LIMIT 10;
"

echo ""
echo "✅ Migration complete!"
