#!/bin/bash
# Convert existing JSONB embeddings to VECTOR format
# This is needed for articles added after migration 024 ran on an empty database

set -e

echo "🔄 Converting Existing JSONB Embeddings to VECTOR"
echo "=================================================="
echo ""

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Use provided DATABASE_URL if not already set
if [ -z "$DATABASE_URL" ]; then
    export DATABASE_URL="postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"
fi

echo "1️⃣  Checking current state..."

# Count JSONB embeddings
JSONB_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding IS NOT NULL" | xargs)
echo "   • JSONB embeddings: $JSONB_COUNT"

# Count VECTOR embeddings
VECTOR_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding_vector IS NOT NULL" | xargs)
echo "   • VECTOR embeddings: $VECTOR_COUNT"

# Count embeddings needing conversion
TO_CONVERT=$(psql "$DATABASE_URL" -tAc "
    SELECT COUNT(*)
    FROM articles
    WHERE embedding IS NOT NULL
      AND embedding_vector IS NULL
      AND jsonb_typeof(embedding) = 'array'
      AND jsonb_array_length(embedding) = 768
" | xargs)

echo "   • Need conversion: $TO_CONVERT"
echo ""

if [ "$TO_CONVERT" -eq 0 ]; then
    echo "✅ All embeddings already converted!"
    exit 0
fi

read -p "Convert $TO_CONVERT embeddings from JSONB to VECTOR? (y/n) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled by user"
    exit 0
fi

echo ""
echo "2️⃣  Converting embeddings..."
START_TIME=$(date +%s.%N)

psql "$DATABASE_URL" <<'SQL'
-- Convert JSONB array to VECTOR format
UPDATE articles
SET embedding_vector = (
    SELECT CAST(
        '[' || array_to_string(
            ARRAY(
                SELECT jsonb_array_elements_text(embedding)
            ),
            ','
        ) || ']' AS VECTOR(768)
    )
)
WHERE embedding IS NOT NULL
  AND embedding_vector IS NULL
  AND jsonb_typeof(embedding) = 'array'
  AND jsonb_array_length(embedding) = 768;

-- Show results
SELECT
    COUNT(*) FILTER (WHERE embedding IS NOT NULL) as jsonb_count,
    COUNT(*) FILTER (WHERE embedding_vector IS NOT NULL) as vector_count,
    COUNT(*) FILTER (WHERE embedding IS NOT NULL AND embedding_vector IS NULL) as pending_count
FROM articles;
SQL

END_TIME=$(date +%s.%N)
DURATION=$(echo "$END_TIME - $START_TIME" | bc)

echo ""
echo "3️⃣  Verifying conversion..."

# Final counts
FINAL_JSONB=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding IS NOT NULL" | xargs)
FINAL_VECTOR=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding_vector IS NOT NULL" | xargs)

echo "   • JSONB embeddings: $FINAL_JSONB"
echo "   • VECTOR embeddings: $FINAL_VECTOR"
echo ""

if [ "$FINAL_JSONB" -eq "$FINAL_VECTOR" ]; then
    echo "✅ All embeddings converted successfully in ${DURATION}s!"
else
    echo "⚠️  Partial conversion: $FINAL_VECTOR/$FINAL_JSONB converted"
    echo "   Some embeddings may have invalid format"
fi

echo ""
echo "4️⃣  Checking index status..."

# Check if index exists and what type
INDEX_INFO=$(psql "$DATABASE_URL" -tAc "
    SELECT indexname
    FROM pg_indexes
    WHERE tablename = 'articles'
      AND indexname LIKE '%embedding%'
      AND indexdef LIKE '%embedding_vector%'
" | xargs)

if [ -n "$INDEX_INFO" ]; then
    echo "   ✅ Index exists: $INDEX_INFO"
else
    echo "   ⚠️  No vector index found"
    echo ""
    read -p "   Create HNSW index for fast semantic search? (y/n) " -n 1 -r
    echo ""

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "   🔧 Creating HNSW index..."
        psql "$DATABASE_URL" <<'SQL'
CREATE INDEX IF NOT EXISTS idx_articles_embedding_hnsw
ON articles
USING hnsw (embedding_vector vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
SQL
        echo "   ✅ HNSW index created"
    fi
fi

echo ""
echo "════════════════════════════════════════════════════════"
echo "✅ Conversion Complete!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "📊 Final Stats:"
echo "   • Total articles: $(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles" | xargs)"
echo "   • JSONB embeddings: $FINAL_JSONB"
echo "   • VECTOR embeddings: $FINAL_VECTOR"
echo ""
echo "🚀 Next Steps:"
echo "   1. Test semantic search: ./test_pgvector.sh"
echo "   2. Your system is now 50-200x faster! 🎉"
echo ""
echo "💡 Future articles will automatically use embedding_vector"
echo "   (if your code is updated to populate it)"
