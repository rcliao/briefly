-- Migration: 005_add_article_themes
-- Description: Add theme relationships and relevance scoring to articles
-- Created: 2025-10-31

-- Add theme_id and relevance_score columns to articles table
ALTER TABLE articles
ADD COLUMN IF NOT EXISTS theme_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS theme_relevance_score FLOAT,
ADD CONSTRAINT fk_articles_theme FOREIGN KEY (theme_id) REFERENCES themes(id) ON DELETE SET NULL;

-- Add index for theme-based queries
CREATE INDEX IF NOT EXISTS idx_articles_theme_id ON articles(theme_id);
CREATE INDEX IF NOT EXISTS idx_articles_theme_relevance ON articles(theme_relevance_score DESC);

-- Add column comments
COMMENT ON COLUMN articles.theme_id IS 'Primary theme assigned to this article';
COMMENT ON COLUMN articles.theme_relevance_score IS 'Relevance score (0.0-1.0) for the assigned theme';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (5, 'Add theme relationships to articles table')
ON CONFLICT (version) DO NOTHING;
