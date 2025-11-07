-- Migration: 019_increase_digest_title_lengths
-- Description: Increase title and tldr_summary length limits for hierarchical summarization
-- Created: 2025-11-06
--
-- Changes:
-- - Increase title from VARCHAR(50) to VARCHAR(200)
-- - Increase tldr_summary from VARCHAR(100) to VARCHAR(250)
-- - Update check constraints to match new limits
--
-- Rationale: Hierarchical summarization generates more descriptive titles that may exceed
-- the original 50-character limit. This provides more room while still enforcing reasonable constraints.

-- Drop existing check constraints
ALTER TABLE digests DROP CONSTRAINT IF EXISTS chk_title_length;
ALTER TABLE digests DROP CONSTRAINT IF EXISTS chk_tldr_length;

-- Increase title column length
ALTER TABLE digests ALTER COLUMN title TYPE VARCHAR(200);

-- Increase tldr_summary column length
ALTER TABLE digests ALTER COLUMN tldr_summary TYPE VARCHAR(250);

-- Add updated check constraints with new limits
ALTER TABLE digests ADD CONSTRAINT chk_title_length
    CHECK (title IS NULL OR (LENGTH(title) >= 10 AND LENGTH(title) <= 200));

ALTER TABLE digests ADD CONSTRAINT chk_tldr_length
    CHECK (tldr_summary IS NULL OR (LENGTH(tldr_summary) >= 30 AND LENGTH(tldr_summary) <= 250));

-- Update comments
COMMENT ON COLUMN digests.title IS 'Digest headline (10-200 chars) - hierarchical summarization may use longer titles';
COMMENT ON COLUMN digests.tldr_summary IS 'One-sentence summary (30-250 chars) - expanded for hierarchical summarization';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (19, 'Increase digest title and tldr_summary length limits for hierarchical summarization')
ON CONFLICT (version) DO NOTHING;
