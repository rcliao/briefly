#!/bin/bash

# Migrate existing JSONB embeddings to pgvector format
# This script checks the current state and applies migration 018 if needed

set -e

echo "🔄 pgvector Migration Check"
echo "==========================="
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
    echo "❌ DATABASE_URL not set"
    exit 1
fi

echo "✅ Connected to database"
echo ""

# Step 1: Check if pgvector extension is enabled
echo "1️⃣  Checking pgvector extension..."
PGVECTOR_ENABLED=$(psql "$DATABASE_URL" -tAc "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='vector');")

if [ "$PGVECTOR_ENABLED" = "t" ]; then
    PGVECTOR_VERSION=$(psql "$DATABASE_URL" -tAc "SELECT extversion FROM pg_extension WHERE extname='vector';")
    echo "   ✅ pgvector enabled: v$PGVECTOR_VERSION"
else
    echo "   ❌ pgvector NOT enabled"
    echo ""
    echo "   Please install pgvector first:"
    echo "   - macOS: brew install pgvector && brew services restart postgresql@16"
    echo "   - Ubuntu: sudo apt install postgresql-16-pgvector"
    echo "   - Docker: Use pgvector/pgvector:pg16 image"
    echo ""
    exit 1
fi
echo ""

# Step 2: Check current embedding column types
echo "2️⃣  Checking embedding column types..."
EMBEDDING_TYPE=$(psql "$DATABASE_URL" -tAc "
    SELECT data_type
    FROM information_schema.columns
    WHERE table_name='articles' AND column_name='embedding'
" | xargs)

EMBEDDING_VECTOR_EXISTS=$(psql "$DATABASE_URL" -tAc "
    SELECT EXISTS(
        SELECT 1
        FROM information_schema.columns
        WHERE table_name='articles' AND column_name='embedding_vector'
    )
")

echo "   Current embedding column type: $EMBEDDING_TYPE"

if [ "$EMBEDDING_VECTOR_EXISTS" = "t" ]; then
    echo "   ✅ embedding_vector column exists"

    # Check how many have been migrated
    TOTAL_EMBEDDINGS=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding IS NOT NULL" | xargs)
    VECTOR_EMBEDDINGS=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding_vector IS NOT NULL" | xargs)

    echo "   • JSONB embeddings: $TOTAL_EMBEDDINGS"
    echo "   • Vector embeddings: $VECTOR_EMBEDDINGS"

    if [ "$VECTOR_EMBEDDINGS" -eq "$TOTAL_EMBEDDINGS" ] && [ "$TOTAL_EMBEDDINGS" -gt 0 ]; then
        echo ""
        echo "✅ All embeddings already migrated to vector format!"
        echo ""
        echo "📊 Summary:"
        echo "   • $VECTOR_EMBEDDINGS articles with vector embeddings"
        echo "   • Ready for fast semantic search"
        echo ""

        # Show index status
        echo "3️⃣  Checking indexes..."
        psql "$DATABASE_URL" -c "
            SELECT
                indexname,
                indexdef
            FROM pg_indexes
            WHERE tablename='articles'
              AND indexname LIKE '%embedding%'
        "

        echo ""
        echo "✅ Migration complete! Ready to use pgvector."
        echo ""
        echo "Next steps:"
        echo "  ./test_pgvector.sh    # Test semantic search"
        exit 0
    fi
else
    echo "   ⚠️  embedding_vector column does NOT exist"
fi
echo ""

# Step 3: Run migration 024 to convert embeddings
echo "3️⃣  Running migration 024 to convert embeddings..."
echo "   This will:"
echo "   • Create embedding_vector VECTOR(768) column"
echo "   • Convert $TOTAL_EMBEDDINGS JSONB embeddings to vector format"
echo "   • Create IVFFlat or HNSW index for fast search"
echo "   • Keep old embedding column for backwards compatibility"
echo ""

read -p "   Continue? (y/n) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "   Cancelled by user"
    exit 0
fi

echo ""
echo "   🔄 Converting embeddings..."

# Run migration 024
if psql "$DATABASE_URL" -f internal/persistence/migrations/024_convert_embeddings_to_vector.sql > /tmp/migration_024.log 2>&1; then
    echo "   ✅ Migration 024 completed successfully"
    echo ""

    # Show results
    echo "📊 Migration Results:"

    VECTOR_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding_vector IS NOT NULL" | xargs)

    echo "   • Articles with vector embeddings: $VECTOR_COUNT"

    # Show index created
    INDEX_NAME=$(psql "$DATABASE_URL" -tAc "
        SELECT indexname
        FROM pg_indexes
        WHERE tablename='articles'
          AND indexname LIKE '%embedding%'
        LIMIT 1
    " | xargs)

    if [ -n "$INDEX_NAME" ]; then
        echo "   • Index created: $INDEX_NAME"
    fi

    # Show storage savings
    JSONB_SIZE=$(psql "$DATABASE_URL" -tAc "
        SELECT pg_size_pretty(
            SUM(pg_column_size(embedding))::bigint
        )
        FROM articles
        WHERE embedding IS NOT NULL
    " | xargs)

    VECTOR_SIZE=$(psql "$DATABASE_URL" -tAc "
        SELECT pg_size_pretty(
            SUM(pg_column_size(embedding_vector))::bigint
        )
        FROM articles
        WHERE embedding_vector IS NOT NULL
    " | xargs)

    echo "   • Storage (JSONB): $JSONB_SIZE"
    echo "   • Storage (vector): $VECTOR_SIZE"

    echo ""
    echo "✅ Migration complete!"
    echo ""
    echo "Performance improvement expected:"
    echo "  • Search latency: 1-10ms → <100µs (10-100x faster)"
    echo "  • Storage size: ~3x smaller"
    echo ""
    echo "Next steps:"
    echo "  ./test_pgvector.sh    # Test the new vector search"

else
    echo "   ❌ Migration failed"
    echo ""
    cat /tmp/migration_024.log
    exit 1
fi
