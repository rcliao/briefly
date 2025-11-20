#!/bin/bash
# Fix Migration 22 - Reset and re-run after adding tag remapping
# This script safely handles the failed migration 22

set -e

echo "🔧 Fixing Migration 22"
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

echo "1️⃣  Checking migration 22 status..."
MIGRATION_22_EXISTS=$(psql "$DATABASE_URL" -tAc "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = 22);")

if [ "$MIGRATION_22_EXISTS" = "t" ]; then
    echo "   ⚠️  Migration 22 was partially applied (recorded but failed)"
    echo "   We need to remove it from schema_migrations and retry"
    echo ""

    read -p "   Remove migration 22 record and retry? (y/n) " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "   Cancelled by user"
        exit 0
    fi

    echo "   🗑️  Removing migration 22 record..."
    psql "$DATABASE_URL" -c "DELETE FROM schema_migrations WHERE version = 22;"
    echo "   ✅ Removed"
else
    echo "   ℹ️  Migration 22 not recorded (good - means it failed before completion)"
fi
echo ""

echo "2️⃣  Checking current theme and tag state..."

# Check how many themes exist
THEME_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM themes;" | xargs)
echo "   • Current themes: $THEME_COUNT"

# Check how many tags reference old themes
OLD_THEME_TAG_COUNT=$(psql "$DATABASE_URL" -tAc "
    SELECT COUNT(*)
    FROM tags
    WHERE theme_id NOT IN ('theme-genai', 'theme-gaming', 'theme-technology', 'theme-healthcare', 'theme-business')
      AND theme_id IS NOT NULL
" | xargs)
echo "   • Tags referencing old themes: $OLD_THEME_TAG_COUNT"

echo ""

echo "3️⃣  Re-running migration 22 with tag remapping..."
./briefly migrate up

echo ""
echo "════════════════════════════════════════════════════════"
echo "✅ Migration 22 Fixed!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "📊 Verification:"
echo ""

# Show final theme count
FINAL_THEME_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM themes;" | xargs)
echo "   Final theme count: $FINAL_THEME_COUNT (should be 5)"

# Show themes
echo ""
echo "   Themes:"
psql "$DATABASE_URL" -c "SELECT id, name FROM themes ORDER BY name;"

# Show tag distribution
echo ""
echo "   Tag distribution by theme:"
psql "$DATABASE_URL" -c "
    SELECT
        COALESCE(t.name, 'No Theme') as theme,
        COUNT(tg.id) as tag_count
    FROM tags tg
    LEFT JOIN themes t ON tg.theme_id = t.id
    GROUP BY t.name
    ORDER BY tag_count DESC;
"

echo ""
echo "🚀 Next Steps:"
echo "   All migrations should now be applied successfully!"
echo "   Run: ./briefly migrate up"
