-- Migration: 013_add_digest_relationships
-- Description: Add many-to-many relationship tables for digests (v2.0)
-- Created: 2025-11-06
--
-- Changes:
-- - Create digest_articles join table (tracks which articles belong to which digests with citation order)
-- - Create digest_themes join table (tracks which themes apply to which digests)
-- - Add indexes for performance on relationship queries

-- ============================================================================
-- DIGEST-ARTICLE RELATIONSHIPS
-- ============================================================================

-- Create digest_articles join table with citation order tracking
CREATE TABLE IF NOT EXISTS digest_articles (
    digest_id VARCHAR(255) NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
    article_id VARCHAR(255) NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    citation_order INTEGER NOT NULL,
    relevance_to_digest FLOAT,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (digest_id, article_id)
);

-- Indexes for digest_articles queries
CREATE INDEX IF NOT EXISTS idx_digest_articles_digest ON digest_articles(digest_id);
CREATE INDEX IF NOT EXISTS idx_digest_articles_article ON digest_articles(article_id);
CREATE INDEX IF NOT EXISTS idx_digest_articles_citation_order ON digest_articles(digest_id, citation_order);
CREATE INDEX IF NOT EXISTS idx_digest_articles_relevance ON digest_articles(relevance_to_digest DESC) WHERE relevance_to_digest IS NOT NULL;

-- Add table and column comments
COMMENT ON TABLE digest_articles IS 'Many-to-many relationship between digests and articles with citation tracking';
COMMENT ON COLUMN digest_articles.digest_id IS 'Reference to the digest';
COMMENT ON COLUMN digest_articles.article_id IS 'Reference to the article';
COMMENT ON COLUMN digest_articles.citation_order IS 'Order in digest for citation numbering ([1], [2], [3], etc.)';
COMMENT ON COLUMN digest_articles.relevance_to_digest IS 'How central this article is to the digest (0.0-1.0, higher = more relevant)';
COMMENT ON COLUMN digest_articles.added_at IS 'When this article was added to the digest';

-- ============================================================================
-- DIGEST-THEME RELATIONSHIPS
-- ============================================================================

-- Create digest_themes join table
-- Note: A digest can belong to multiple themes (e.g., "GenAI" + "Cloud")
CREATE TABLE IF NOT EXISTS digest_themes (
    digest_id VARCHAR(255) NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
    theme_id VARCHAR(255) NOT NULL REFERENCES themes(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (digest_id, theme_id)
);

-- Indexes for digest_themes queries (critical for theme filtering on homepage)
CREATE INDEX IF NOT EXISTS idx_digest_themes_digest ON digest_themes(digest_id);
CREATE INDEX IF NOT EXISTS idx_digest_themes_theme ON digest_themes(theme_id);

-- Composite index for common query pattern: filter digests by theme + date
CREATE INDEX IF NOT EXISTS idx_digest_themes_theme_digest ON digest_themes(theme_id, digest_id);

-- Add table and column comments
COMMENT ON TABLE digest_themes IS 'Many-to-many relationship between digests and themes for filtering';
COMMENT ON COLUMN digest_themes.digest_id IS 'Reference to the digest';
COMMENT ON COLUMN digest_themes.theme_id IS 'Reference to the theme';
COMMENT ON COLUMN digest_themes.added_at IS 'When this theme was assigned to the digest';

-- ============================================================================
-- CONSTRAINTS AND VALIDATIONS
-- ============================================================================

-- Ensure citation_order is positive and reasonable (max 100 articles per digest)
ALTER TABLE digest_articles ADD CONSTRAINT digest_articles_citation_order_check
    CHECK (citation_order > 0 AND citation_order <= 100);

-- Ensure relevance_to_digest is between 0 and 1 if specified
ALTER TABLE digest_articles ADD CONSTRAINT digest_articles_relevance_check
    CHECK (relevance_to_digest IS NULL OR (relevance_to_digest >= 0 AND relevance_to_digest <= 1));

-- ============================================================================
-- RECORD MIGRATION
-- ============================================================================

INSERT INTO schema_migrations (version, description)
VALUES (13, 'Add digest_articles and digest_themes relationship tables (v2.0)')
ON CONFLICT (version) DO NOTHING;
