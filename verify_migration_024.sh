#!/bin/bash
# Verify Migration 024 Results
# This checks what actually happened during the migration

set -e

echo "🔍 Migration 024 Verification"
echo "=============================="
echo ""

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Use provided DATABASE_URL if not already set
if [ -z "$DATABASE_URL" ]; then
    export DATABASE_URL="postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"
fi

echo "1️⃣  Checking if pgvector extension is enabled..."
PGVECTOR_ENABLED=$(psql "$DATABASE_URL" -tAc "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='vector');")

if [ "$PGVECTOR_ENABLED" = "t" ]; then
    PGVECTOR_VERSION=$(psql "$DATABASE_URL" -tAc "SELECT extversion FROM pg_extension WHERE extname='vector';")
    echo "   ✅ pgvector enabled: v$PGVECTOR_VERSION"
else
    echo "   ❌ pgvector NOT enabled"
fi
echo ""

echo "2️⃣  Checking if embedding_vector column exists..."
VECTOR_COLUMN_EXISTS=$(psql "$DATABASE_URL" -tAc "
    SELECT EXISTS(
        SELECT 1
        FROM information_schema.columns
        WHERE table_name='articles' AND column_name='embedding_vector'
    )
")

if [ "$VECTOR_COLUMN_EXISTS" = "t" ]; then
    echo "   ✅ embedding_vector column EXISTS"

    # Count migrated embeddings
    VECTOR_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding_vector IS NOT NULL" | xargs)
    echo "   • Vector embeddings: $VECTOR_COUNT"
else
    echo "   ❌ embedding_vector column does NOT exist"
    echo "   ℹ️  This is expected if pgvector is not installed"
fi
echo ""

echo "3️⃣  Checking JSONB embeddings..."
JSONB_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding IS NOT NULL" | xargs)
echo "   • JSONB embeddings: $JSONB_COUNT"
echo ""

echo "4️⃣  Checking migration 024 status..."
psql "$DATABASE_URL" -c "
    SELECT version, description, applied_at
    FROM schema_migrations
    WHERE version = 24
"
echo ""

echo "5️⃣  Summary"
echo "=========="
if [ "$PGVECTOR_ENABLED" = "t" ]; then
    echo "✅ pgvector is enabled - migration should have converted embeddings"
    if [ "$VECTOR_COLUMN_EXISTS" = "t" ]; then
        echo "✅ embedding_vector column created successfully"
        echo "✅ Migration 024 completed full conversion"
    else
        echo "⚠️  embedding_vector column missing - something went wrong"
    fi
else
    echo "⚠️  pgvector NOT enabled - migration gracefully skipped conversion"
    if [ "$VECTOR_COLUMN_EXISTS" = "t" ]; then
        echo "⚠️  Unexpected: embedding_vector column exists despite pgvector being disabled"
    else
        echo "✅ No vector column created - graceful skip worked correctly"
    fi
    echo ""
    echo "📝 System is using JSONB embeddings (slower but functional)"
    echo "   To enable pgvector performance boost:"
    echo "   1. Install: brew install pgvector && brew services restart postgresql@16"
    echo "   2. Enable: psql \$DATABASE_URL -c 'CREATE EXTENSION vector;'"
    echo "   3. Re-run: ./briefly migrate down 24 && ./briefly migrate up"
fi
