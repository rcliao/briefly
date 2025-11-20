-- Setup pgvector for semantic search
-- Run with: psql $DATABASE_URL -f setup_pgvector.sql

-- 1. Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. Check if embedding_vector column exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'articles' AND column_name = 'embedding_vector'
    ) THEN
        RAISE NOTICE 'embedding_vector column does not exist - will be created by migration 018';
    ELSE
        RAISE NOTICE 'embedding_vector column already exists';
    END IF;
END $$;

-- Show current status
SELECT
    COUNT(*) FILTER (WHERE embedding IS NOT NULL) as jsonb_embeddings,
    COUNT(*) FILTER (WHERE embedding_vector IS NOT NULL) as vector_embeddings,
    COUNT(*) as total_articles
FROM articles;
