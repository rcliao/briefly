#!/bin/bash
# Reset articles and re-aggregate to test full pipeline with pgvector
# This verifies that new articles automatically populate embedding_vector

set -e

echo "🔄 Reset and Re-aggregate Articles"
echo "==================================="
echo ""

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Use provided DATABASE_URL if not already set
if [ -z "$DATABASE_URL" ]; then
    export DATABASE_URL="postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"
fi

echo "⚠️  WARNING: This will delete all existing articles and digests!"
echo ""

# Show current state
ARTICLE_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles" | xargs)
DIGEST_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM digests" | xargs)
FEED_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM feeds" | xargs)

echo "Current state:"
echo "  • Articles: $ARTICLE_COUNT"
echo "  • Digests: $DIGEST_COUNT"
echo "  • Feeds: $FEED_COUNT"
echo ""

read -p "Delete all articles and digests? (y/n) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled by user"
    exit 0
fi

echo ""
echo "1️⃣  Deleting articles and digests..."

psql "$DATABASE_URL" <<'SQL'
-- Delete in correct order (respecting foreign keys)
DELETE FROM digest_articles;
DELETE FROM digest_themes;
DELETE FROM article_tags;
DELETE FROM citations;
DELETE FROM digests;
DELETE FROM summaries;
DELETE FROM articles;

-- Show counts
SELECT
    (SELECT COUNT(*) FROM articles) as articles,
    (SELECT COUNT(*) FROM digests) as digests,
    (SELECT COUNT(*) FROM summaries) as summaries;
SQL

echo "   ✅ Deleted all articles and digests"
echo ""

echo "2️⃣  Current feeds:"
psql "$DATABASE_URL" -c "SELECT id, title, url, active FROM feeds ORDER BY title;"
echo ""

if [ "$FEED_COUNT" -eq 0 ]; then
    echo "⚠️  No feeds configured!"
    echo ""
    echo "Add some feeds first:"
    echo "  ./briefly feed add https://hnrss.org/newest"
    echo "  ./briefly feed add https://blog.golang.org/feed.atom"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled by user"
        exit 0
    fi
fi

echo ""
echo "3️⃣  Re-aggregating articles..."
echo "   This will fetch articles from feeds and create embeddings"
echo ""

# Make sure we have the latest build
if [ ! -f "./briefly" ]; then
    echo "   Building briefly..."
    make build
fi

# Aggregate articles
./briefly aggregate --since 24

echo ""
echo "4️⃣  Verifying results..."

# Count results
NEW_ARTICLE_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles" | xargs)
JSONB_EMBEDDING_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding IS NOT NULL" | xargs)
VECTOR_EMBEDDING_COUNT=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM articles WHERE embedding_vector IS NOT NULL" | xargs)

echo "   • Total articles: $NEW_ARTICLE_COUNT"
echo "   • JSONB embeddings: $JSONB_EMBEDDING_COUNT"
echo "   • VECTOR embeddings: $VECTOR_EMBEDDING_COUNT"
echo ""

if [ "$VECTOR_EMBEDDING_COUNT" -eq "$NEW_ARTICLE_COUNT" ] && [ "$NEW_ARTICLE_COUNT" -gt 0 ]; then
    echo "✅ Success! All articles have VECTOR embeddings!"
    echo ""
    echo "📊 Database state:"
    psql "$DATABASE_URL" -c "
        SELECT
            COUNT(*) as total_articles,
            COUNT(*) FILTER (WHERE embedding IS NOT NULL) as jsonb_embeddings,
            COUNT(*) FILTER (WHERE embedding_vector IS NOT NULL) as vector_embeddings
        FROM articles;
    "
    echo ""
    echo "🚀 Next steps:"
    echo "   1. Test semantic search: ./test_pgvector.sh"
    echo "   2. Generate digest: ./briefly digest generate --since 7"
elif [ "$NEW_ARTICLE_COUNT" -eq 0 ]; then
    echo "⚠️  No articles were aggregated"
    echo ""
    echo "Possible reasons:"
    echo "  • No feeds configured (add feeds with: ./briefly feed add <url>)"
    echo "  • No new articles in last 24 hours (try --since 168 for 1 week)"
    echo "  • Feed fetch errors (check logs)"
else
    echo "⚠️  Partial success:"
    echo "   • Articles created: $NEW_ARTICLE_COUNT"
    echo "   • VECTOR embeddings: $VECTOR_EMBEDDING_COUNT"
    echo ""
    echo "This means the code update may not be working correctly."
    echo "Check internal/persistence/postgres.go CreateOrUpdate method"
fi

echo ""
echo "════════════════════════════════════════════════════════"
echo "Done!"
echo "════════════════════════════════════════════════════════"
