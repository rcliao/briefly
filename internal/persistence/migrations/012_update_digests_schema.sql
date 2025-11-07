-- Migration: 012_update_digests_schema
-- Description: Update digests table for many-digests architecture (v2.0)
-- Created: 2025-11-06
--
-- Changes:
-- - Add summary field (markdown with [[N]](url) citations)
-- - Add key_moments field (JSONB array of quote objects)
-- - Add perspectives field (JSONB array of supporting/opposing viewpoints)
-- - Add cluster_id field (HDBSCAN cluster reference)
-- - Add processed_date field (DATE for daily/weekly queries)
-- - Add article_count field (performance optimization)
-- - Remove date UNIQUE constraint (multiple digests per day)
-- - Migrate data from content JSONB to new fields

-- Add new columns (all nullable initially for data migration)
ALTER TABLE digests ADD COLUMN IF NOT EXISTS summary TEXT;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS key_moments JSONB;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS perspectives JSONB;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS cluster_id INTEGER;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS processed_date DATE;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS article_count INTEGER DEFAULT 0;

-- Remove UNIQUE constraint on date (multiple digests per day in v2.0)
ALTER TABLE digests DROP CONSTRAINT IF EXISTS digests_date_key;

-- Migrate existing data from content JSONB to new fields
-- Note: This is a best-effort migration for existing digests
UPDATE digests SET
    summary = COALESCE(content->>'summary', content->>'content', 'Legacy digest - content not migrated'),
    processed_date = date,
    article_count = COALESCE((content->'metadata'->>'article_count')::INTEGER, 0),
    cluster_id = 0  -- Legacy digests treated as cluster 0
WHERE summary IS NULL;

-- Add constraints after data migration
ALTER TABLE digests ALTER COLUMN summary SET NOT NULL;
ALTER TABLE digests ALTER COLUMN processed_date SET NOT NULL;
ALTER TABLE digests ALTER COLUMN article_count SET NOT NULL;
-- cluster_id can be NULL for weekly digests that aggregate multiple clusters

-- Add indexes for common queries
CREATE INDEX IF NOT EXISTS idx_digests_processed_date ON digests(processed_date DESC);
CREATE INDEX IF NOT EXISTS idx_digests_cluster_id ON digests(cluster_id) WHERE cluster_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_digests_article_count ON digests(article_count DESC);

-- Add GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_digests_key_moments ON digests USING GIN (key_moments);
CREATE INDEX IF NOT EXISTS idx_digests_perspectives ON digests USING GIN (perspectives);

-- Add column comments for documentation
COMMENT ON COLUMN digests.summary IS 'Markdown summary with [[N]](url) inline citations (2-3 paragraphs)';
COMMENT ON COLUMN digests.tldr_summary IS 'One-sentence summary (50-70 chars ideal for UI)';
COMMENT ON COLUMN digests.key_moments IS 'Array of {quote: string, citation_number: int} objects';
COMMENT ON COLUMN digests.perspectives IS 'Array of {type: "supporting"|"opposing", summary: string, citation_numbers: [int]} objects';
COMMENT ON COLUMN digests.cluster_id IS 'HDBSCAN cluster ID this digest represents (NULL for weekly aggregated digests, -1 for noise)';
COMMENT ON COLUMN digests.processed_date IS 'Date when this digest was generated (for daily/weekly filtering)';
COMMENT ON COLUMN digests.article_count IS 'Number of articles in this digest (cached for performance)';

-- Update table comment
COMMENT ON TABLE digests IS 'Generated news digests - one digest per topic cluster (v2.0 many-digests architecture)';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (12, 'Update digests table for many-digests architecture (v2.0)')
ON CONFLICT (version) DO NOTHING;
