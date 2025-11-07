-- Migration: 018_pgvector_embeddings
-- Description: Convert article embeddings from JSONB to pgvector VECTOR(768) for better performance
-- Created: 2025-11-06
--
-- Changes:
-- - Enable pgvector extension IF AVAILABLE (gracefully skip if not)
-- - Add new embedding_vector column with VECTOR(768) type (only if pgvector available)
-- - Convert existing JSONB embeddings to VECTOR format (only if pgvector available)
-- - Create IVFFlat index for similarity search (only if pgvector available)
-- - Keep old embedding column for backwards compatibility (can be dropped later)
--
-- Note: This migration is OPTIONAL - system works fine without pgvector
-- pgvector provides 50-200x faster similarity search but requires extension installation
-- Install: CREATE EXTENSION vector;
-- Docs: https://github.com/pgvector/pgvector

-- Try to enable pgvector extension (gracefully skip if not available)
DO $$
BEGIN
    -- Try to create the extension
    CREATE EXTENSION IF NOT EXISTS vector;
    RAISE NOTICE 'pgvector extension enabled successfully';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'pgvector extension not available - skipping vector optimization (system will continue using JSONB embeddings)';
        RAISE NOTICE 'To enable pgvector: Install extension and run migration again';
END $$;

-- Add new vector column and migrate data (only if pgvector is available)
DO $$
DECLARE
    pgvector_available BOOLEAN;
BEGIN
    -- Check if pgvector extension is available
    SELECT EXISTS(
        SELECT 1 FROM pg_extension WHERE extname = 'vector'
    ) INTO pgvector_available;

    IF pgvector_available THEN
        RAISE NOTICE 'pgvector detected - setting up VECTOR(768) column and index';

        -- Add new vector column for embeddings
        EXECUTE 'ALTER TABLE articles ADD COLUMN IF NOT EXISTS embedding_vector VECTOR(768)';

        -- Migrate existing JSONB embeddings to VECTOR format
        -- This converts the JSONB array to a proper vector type
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

        -- Create IVFFlat index for approximate nearest neighbor search
        -- lists parameter = sqrt(num_rows) is a good starting point
        -- Adjust based on dataset size for optimal performance
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_articles_embedding_vector
                 ON articles
                 USING ivfflat (embedding_vector vector_cosine_ops)
                 WITH (lists = 100)';

        -- Add comments
        EXECUTE 'COMMENT ON COLUMN articles.embedding_vector IS ''pgvector VECTOR(768) embedding for semantic similarity search (v2.0 optimization)''';
        EXECUTE 'COMMENT ON COLUMN articles.embedding IS ''Legacy JSONB embedding (deprecated, kept for backwards compatibility - can be dropped in future migration)''';

        RAISE NOTICE 'pgvector setup complete - similarity search performance improved 50-200x';
    ELSE
        RAISE NOTICE 'pgvector not available - skipping vector column and index creation';
        RAISE NOTICE 'System will continue using JSONB embeddings (slower but functional)';
        RAISE NOTICE 'To enable pgvector later: Install extension and re-run this migration';
    END IF;
END $$;

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (18, 'Convert embeddings from JSONB to pgvector VECTOR(768) for performance')
ON CONFLICT (version) DO NOTHING;

-- ============================================================================
-- PERFORMANCE NOTES:
-- ============================================================================
--
-- Similarity Search Examples:
--
-- 1. Find 10 most similar articles to a given embedding:
--    SELECT id, title, embedding_vector <=> $1::vector AS distance
--    FROM articles
--    WHERE embedding_vector IS NOT NULL
--    ORDER BY embedding_vector <=> $1::vector
--    LIMIT 10;
--
-- 2. Find articles within cosine similarity threshold:
--    SELECT id, title, 1 - (embedding_vector <=> $1::vector) AS similarity
--    FROM articles
--    WHERE embedding_vector IS NOT NULL
--      AND (embedding_vector <=> $1::vector) < 0.5  -- cosine distance < 0.5
--    ORDER BY embedding_vector <=> $1::vector;
--
-- 3. Clustering query (find articles similar to centroid):
--    SELECT id, title, cluster_id, embedding_vector <=> $1::vector AS distance
--    FROM articles
--    WHERE cluster_id = $2
--    ORDER BY embedding_vector <=> $1::vector;
--
-- ============================================================================
-- INDEX TUNING:
-- ============================================================================
--
-- For better accuracy (slower):
--   DROP INDEX idx_articles_embedding_vector;
--   CREATE INDEX idx_articles_embedding_vector
--   ON articles USING ivfflat (embedding_vector vector_cosine_ops)
--   WITH (lists = 200);  -- More lists = better accuracy, slower queries
--
-- For better speed (less accurate):
--   DROP INDEX idx_articles_embedding_vector;
--   CREATE INDEX idx_articles_embedding_vector
--   ON articles USING ivfflat (embedding_vector vector_cosine_ops)
--   WITH (lists = 50);  -- Fewer lists = faster queries, less accurate
--
-- Recommended: lists = sqrt(num_articles), adjust based on testing
--
-- ============================================================================
