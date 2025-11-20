#!/bin/bash
# Fix Migration 19 - Reset and re-run after fixing theme_id references
# This script safely handles the failed migration 19

set -e

echo "🔧 Fixing Migration 19"
echo "====================="
echo ""

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Use provided DATABASE_URL if not already set
if [ -z "$DATABASE_URL" ]; then
    export DATABASE_URL="postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"
fi

echo "1️⃣  Checking migration 19 status..."
MIGRATION_19_EXISTS=$(psql "$DATABASE_URL" -tAc "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = 19);")

if [ "$MIGRATION_19_EXISTS" = "t" ]; then
    echo "   ⚠️  Migration 19 was partially applied (recorded but failed)"
    echo "   We need to remove it from schema_migrations and retry"
    echo ""

    read -p "   Remove migration 19 record and retry? (y/n) " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "   Cancelled by user"
        exit 0
    fi

    echo "   🗑️  Removing migration 19 record..."
    psql "$DATABASE_URL" -c "DELETE FROM schema_migrations WHERE version = 19;"
    echo "   ✅ Removed"
else
    echo "   ℹ️  Migration 19 not recorded (good - means it failed before completion)"
fi
echo ""

echo "2️⃣  Checking for partial tables from migration 19..."

# Check if tags table exists
TAGS_EXISTS=$(psql "$DATABASE_URL" -tAc "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='tags');")

if [ "$TAGS_EXISTS" = "t" ]; then
    echo "   ⚠️  tags table exists (partial migration)"
    echo "   We need to drop it and let migration 19 recreate it"
    echo ""

    read -p "   Drop tags and article_tags tables? (y/n) " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "   Cancelled by user"
        exit 0
    fi

    echo "   🗑️  Dropping tables..."
    psql "$DATABASE_URL" -c "DROP TABLE IF EXISTS article_tags CASCADE;"
    psql "$DATABASE_URL" -c "DROP TABLE IF EXISTS tags CASCADE;"
    echo "   ✅ Dropped"
else
    echo "   ℹ️  tags table doesn't exist (good)"
fi
echo ""

echo "3️⃣  Re-running migration 19 with fixed theme references..."
./briefly migrate up

echo ""
echo "════════════════════════════════════════════════════════"
echo "✅ Migration 19 Fixed!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "📊 Verification:"
echo "   • Check tags table:"
echo "     psql \$DATABASE_URL -c 'SELECT COUNT(*) FROM tags;'"
echo ""
echo "   • View theme distribution:"
echo "     psql \$DATABASE_URL -c 'SELECT theme_id, COUNT(*) FROM tags GROUP BY theme_id ORDER BY COUNT(*) DESC;'"
echo ""
echo "🚀 Next Steps:"
echo "   Continue with your work - all migrations should now be applied!"
