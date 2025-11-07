-- Migration: 014_update_articles_and_citations
-- Description: Update articles table (publisher) and citations table for digest citations (v2.0)
-- Created: 2025-11-06
--
-- Changes:
-- - Add publisher field to articles table (for UI display)
-- - Update citations table to support digest citations with citation numbers
-- - Remove UNIQUE constraint on citations.article_id (multiple digests can cite same article)
-- - Add digest_id and citation_number fields to citations
-- - Optionally: Prepare for pgvector migration (commented out, requires extension)

-- ============================================================================
-- ARTICLES TABLE UPDATES
-- ============================================================================

-- Add publisher field to articles (extracted from URL domain)
ALTER TABLE articles ADD COLUMN IF NOT EXISTS publisher VARCHAR(255);

-- Create index for publisher queries (useful for publisher filtering in UI)
CREATE INDEX IF NOT EXISTS idx_articles_publisher ON articles(publisher);

-- Add column comment
COMMENT ON COLUMN articles.publisher IS 'Publisher domain extracted from URL (e.g., "anthropic.com", "openai.com", "techcrunch.com")';

-- ============================================================================
-- CITATIONS TABLE UPDATES
-- ============================================================================

-- Drop UNIQUE constraint on article_id (multiple digests can cite same article)
ALTER TABLE citations DROP CONSTRAINT IF EXISTS citations_article_id_key;

-- Add new columns for digest citations
ALTER TABLE citations ADD COLUMN IF NOT EXISTS digest_id VARCHAR(255) REFERENCES digests(id) ON DELETE CASCADE;
ALTER TABLE citations ADD COLUMN IF NOT EXISTS citation_number INTEGER;
ALTER TABLE citations ADD COLUMN IF NOT EXISTS context TEXT;

-- Update existing citations to have NULL digest_id (legacy article metadata citations)
-- New citations will have digest_id set (digest citation references)

-- Create indexes for citation queries
CREATE INDEX IF NOT EXISTS idx_citations_digest_id ON citations(digest_id) WHERE digest_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_citations_digest_citation ON citations(digest_id, citation_number) WHERE digest_id IS NOT NULL;

-- Add new column comments
COMMENT ON COLUMN citations.digest_id IS 'Reference to digest (NULL for legacy article metadata, NOT NULL for digest citations)';
COMMENT ON COLUMN citations.citation_number IS 'Citation number in digest summary ([1], [2], [3], etc.)';
COMMENT ON COLUMN citations.context IS 'Surrounding text where citation appears in digest summary';

-- Update table comment
COMMENT ON TABLE citations IS 'Source metadata and inline citation tracking (both article metadata and digest citations)';

-- ============================================================================
-- PGVECTOR MIGRATION (OPTIONAL - REQUIRES EXTENSION)
-- ============================================================================

-- Uncomment these lines after enabling pgvector extension:
-- CREATE EXTENSION IF NOT EXISTS vector;
--
-- -- Convert embedding column from JSONB to VECTOR(768)
-- -- WARNING: This is a breaking change that requires data migration
-- ALTER TABLE articles ADD COLUMN embedding_vector VECTOR(768);
--
-- -- Migrate existing embeddings from JSONB to VECTOR
-- UPDATE articles SET embedding_vector =
--     CAST(ARRAY(SELECT jsonb_array_elements_text(embedding)) AS FLOAT8[])::VECTOR(768)
-- WHERE embedding IS NOT NULL;
--
-- -- Drop old JSONB column and rename new column
-- ALTER TABLE articles DROP COLUMN embedding;
-- ALTER TABLE articles RENAME COLUMN embedding_vector TO embedding;
--
-- -- Create IVFFlat index for approximate nearest neighbor search
-- CREATE INDEX idx_articles_embedding ON articles USING ivfflat (embedding vector_cosine_ops)
--     WITH (lists = 100);  -- lists parameter: sqrt(num_articles) typically
--
-- COMMENT ON COLUMN articles.embedding IS '768-dimensional semantic vector from Gemini text-embedding-004 (pgvector)';

-- ============================================================================
-- CONSTRAINTS AND VALIDATIONS
-- ============================================================================

-- Ensure citation_number is positive if specified
ALTER TABLE citations ADD CONSTRAINT citations_citation_number_check
    CHECK (citation_number IS NULL OR (citation_number > 0 AND citation_number <= 100));

-- Ensure digest citations have both digest_id and citation_number
-- (Either both NULL for article metadata, or both NOT NULL for digest citations)
ALTER TABLE citations ADD CONSTRAINT citations_digest_fields_check
    CHECK (
        (digest_id IS NULL AND citation_number IS NULL) OR
        (digest_id IS NOT NULL AND citation_number IS NOT NULL)
    );

-- ============================================================================
-- RECORD MIGRATION
-- ============================================================================

INSERT INTO schema_migrations (version, description)
VALUES (14, 'Update articles (publisher) and citations (digest citations) for v2.0')
ON CONFLICT (version) DO NOTHING;
