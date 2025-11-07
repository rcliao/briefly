-- Migration: 016_add_publisher_field
-- Description: Add publisher domain field to articles table for better article display
-- Created: 2025-11-06
--
-- Changes:
-- - Add publisher VARCHAR(255) column to articles table
-- - Add index on publisher for filtering/grouping queries
-- - Extract publisher from existing article URLs and populate field

-- Add publisher column to articles table
ALTER TABLE articles ADD COLUMN IF NOT EXISTS publisher VARCHAR(255);

-- Add index for publisher queries (group by publisher, filter by publisher)
CREATE INDEX IF NOT EXISTS idx_articles_publisher ON articles(publisher);

-- Extract and populate publisher from existing URLs
-- This updates articles to extract the domain from the URL as publisher
UPDATE articles
SET publisher = SUBSTRING(url FROM 'https?://([^/]+)')
WHERE publisher IS NULL AND url IS NOT NULL;

-- Add comment explaining field purpose
COMMENT ON COLUMN articles.publisher IS 'Publisher domain extracted from URL (e.g., "anthropic.com", "openai.com") - used for display and grouping';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (16, 'Add publisher field to articles table')
ON CONFLICT (version) DO NOTHING;
