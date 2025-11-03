-- Migration: 008_add_citations
-- Description: Add citations table for source attribution and metadata
-- Created: 2025-11-03

-- Citations table stores metadata about article sources for proper attribution
CREATE TABLE IF NOT EXISTS citations (
    id VARCHAR(255) PRIMARY KEY,
    article_id VARCHAR(255) NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title TEXT,
    publisher VARCHAR(255),
    author VARCHAR(255),
    published_date TIMESTAMP WITH TIME ZONE,
    accessed_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(article_id)  -- One citation per article
);

CREATE INDEX IF NOT EXISTS idx_citations_article_id ON citations(article_id);
CREATE INDEX IF NOT EXISTS idx_citations_publisher ON citations(publisher);
CREATE INDEX IF NOT EXISTS idx_citations_published_date ON citations(published_date DESC);
CREATE INDEX IF NOT EXISTS idx_citations_accessed_date ON citations(accessed_date DESC);
CREATE INDEX IF NOT EXISTS idx_citations_url ON citations(url);

-- GIN index for metadata JSONB queries
CREATE INDEX IF NOT EXISTS idx_citations_metadata ON citations USING GIN (metadata);

-- Add table comments
COMMENT ON TABLE citations IS 'Source metadata and attribution information for articles';
COMMENT ON COLUMN citations.article_id IS 'Reference to the source article';
COMMENT ON COLUMN citations.url IS 'Canonical URL of the source';
COMMENT ON COLUMN citations.title IS 'Original title from the source';
COMMENT ON COLUMN citations.publisher IS 'Publisher or domain name';
COMMENT ON COLUMN citations.author IS 'Article author if available';
COMMENT ON COLUMN citations.published_date IS 'Original publication date from source';
COMMENT ON COLUMN citations.accessed_date IS 'When we fetched this article';
COMMENT ON COLUMN citations.metadata IS 'Additional metadata (e.g., DOI, ISBN, issue number) as JSON';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (8, 'Add citations table for source attribution and metadata')
ON CONFLICT (version) DO NOTHING;
