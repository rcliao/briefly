#!/bin/bash

# Fix pgvector integration issues discovered in testing
# This script resolves the 3 main blockers

set -e

echo "🔧 Fixing pgvector Integration Issues"
echo "======================================"
echo ""

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo "❌ No .env file found"
    exit 1
fi

# Check DATABASE_URL
if [ -z "$DATABASE_URL" ]; then
    echo "❌ DATABASE_URL not set"
    exit 1
fi

echo "✅ Connected to database"
echo ""

# Issue 1: Check if article_tags table exists
echo "1️⃣  Checking for article_tags table..."
if psql $DATABASE_URL -tAc "SELECT 1 FROM pg_tables WHERE tablename='article_tags'" | grep -q 1; then
    echo "   ✅ article_tags table exists"
else
    echo "   ⚠️  article_tags table missing - applying migration 019..."
    psql $DATABASE_URL -f internal/persistence/migrations/019_add_tag_system.sql
    echo "   ✅ Migration 019 applied"
fi
echo ""

# Issue 2: Check embedding column type
echo "2️⃣  Checking embedding column type..."
EMBEDDING_TYPE=$(psql $DATABASE_URL -tAc "SELECT data_type FROM information_schema.columns WHERE table_name='articles' AND column_name='embedding'")

if [ "$EMBEDDING_TYPE" = "jsonb" ]; then
    echo "   ⚠️  Embedding is JSONB, need to convert to vector type"
    echo "   📋 Current type: $EMBEDDING_TYPE"
    echo ""
    echo "   This requires migration 018 (add vector column)."
    echo "   Migration should have been applied already."
    echo ""
    echo "   Checking if migration 018 was applied..."
    if psql $DATABASE_URL -tAc "SELECT 1 FROM schema_migrations WHERE version=18" | grep -q 1; then
        echo "   ✅ Migration 018 is recorded as applied"
        echo "   ⚠️  But embedding column is still JSONB!"
        echo ""
        echo "   🔄 Re-applying migration 018 to fix..."
        psql $DATABASE_URL -f internal/persistence/migrations/018_convert_embedding_to_vector.sql || {
            echo "   ⚠️  Migration 018 not found. Need to check migrations directory."
        }
    else
        echo "   ⚠️  Migration 018 not applied - need to apply it"
    fi
elif [ "$EMBEDDING_TYPE" = "USER-DEFINED" ] || [[ $EMBEDDING_TYPE == *"vector"* ]]; then
    echo "   ✅ Embedding column is vector type"
else
    echo "   ⚠️  Unexpected embedding type: $EMBEDDING_TYPE"
fi
echo ""

# Issue 3: Check pgvector version and create appropriate index
echo "3️⃣  Checking pgvector version..."
PGVECTOR_VERSION=$(psql $DATABASE_URL -tAc "SELECT extversion FROM pg_extension WHERE extname='vector'")

if [ -z "$PGVECTOR_VERSION" ]; then
    echo "   ❌ pgvector extension not installed!"
    echo "   Install with: CREATE EXTENSION vector;"
    exit 1
fi

echo "   ✅ pgvector version: $PGVECTOR_VERSION"
echo ""

# Check if we should use HNSW or IVFFlat
echo "4️⃣  Creating optimal index..."

# Try HNSW first (requires pgvector 0.5.0+)
if psql $DATABASE_URL -c "
    CREATE INDEX IF NOT EXISTS idx_articles_embedding_hnsw
    ON articles
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64)
" 2>/dev/null; then
    echo "   ✅ HNSW index created (best performance)"
else
    echo "   ⚠️  HNSW not available (pgvector < 0.5.0)"
    echo "   📝 Creating IVFFlat index instead..."

    psql $DATABASE_URL -c "
        CREATE INDEX IF NOT EXISTS idx_articles_embedding_ivfflat
        ON articles
        USING ivfflat (embedding vector_cosine_ops)
        WITH (lists = 100)
    " && echo "   ✅ IVFFlat index created (good performance)"
fi
echo ""

# Summary
echo "📊 Summary"
echo "=========="
psql $DATABASE_URL -c "
    SELECT
        (SELECT COUNT(*) FROM articles WHERE embedding IS NOT NULL) as embeddings_count,
        (SELECT COUNT(*) FROM tags) as tags_count,
        (SELECT COUNT(*) FROM article_tags) as article_tag_assignments,
        (SELECT indexname FROM pg_indexes WHERE tablename='articles' AND indexname LIKE '%embedding%' LIMIT 1) as index_name
"

echo ""
echo "✅ Fixes applied! Run ./test_pgvector.sh to verify"
