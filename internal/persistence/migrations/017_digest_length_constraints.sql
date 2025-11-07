-- Migration: 017_digest_length_constraints
-- Description: Add length constraints to digest title and tldr_summary fields for v2.0 design compliance
-- Created: 2025-11-06
--
-- Changes:
-- - Add title column if it doesn't exist (VARCHAR(50))
-- - Change tldr_summary to VARCHAR(100) if it exists, add if it doesn't
-- - Truncate any existing values that exceed new limits
-- - Add check constraints to enforce limits at database level

-- Add title column if it doesn't exist
ALTER TABLE digests ADD COLUMN IF NOT EXISTS title VARCHAR(50);

-- Add tldr_summary column if it doesn't exist
ALTER TABLE digests ADD COLUMN IF NOT EXISTS tldr_summary VARCHAR(100);

-- Truncate existing titles that are too long (if any data exists)
UPDATE digests
SET title = SUBSTRING(title FROM 1 FOR 50)
WHERE title IS NOT NULL AND LENGTH(title) > 50;

-- Truncate existing TLDR summaries that are too long (if any data exists)
UPDATE digests
SET tldr_summary = SUBSTRING(tldr_summary FROM 1 FOR 100)
WHERE tldr_summary IS NOT NULL AND LENGTH(tldr_summary) > 100;

-- If column already exists as TEXT, alter it to VARCHAR
-- This is done by checking information_schema
DO $$
BEGIN
    -- Alter title column type if it exists and is TEXT
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'digests' AND column_name = 'title'
        AND data_type = 'text'
    ) THEN
        ALTER TABLE digests ALTER COLUMN title TYPE VARCHAR(50);
    END IF;

    -- Alter tldr_summary column type if it exists and is TEXT
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'digests' AND column_name = 'tldr_summary'
        AND data_type = 'text'
    ) THEN
        ALTER TABLE digests ALTER COLUMN tldr_summary TYPE VARCHAR(100);
    END IF;
END $$;

-- Add check constraints for additional validation (drop if exists first)
ALTER TABLE digests DROP CONSTRAINT IF EXISTS chk_title_length;
ALTER TABLE digests DROP CONSTRAINT IF EXISTS chk_tldr_length;

ALTER TABLE digests ADD CONSTRAINT chk_title_length
    CHECK (title IS NULL OR (LENGTH(title) >= 10 AND LENGTH(title) <= 50));

ALTER TABLE digests ADD CONSTRAINT chk_tldr_length
    CHECK (tldr_summary IS NULL OR (LENGTH(tldr_summary) >= 30 AND LENGTH(tldr_summary) <= 100));

-- Add comments explaining constraints
COMMENT ON COLUMN digests.title IS 'Digest headline (10-50 chars) - v2.0 design guideline: 30-50 chars ideal';
COMMENT ON COLUMN digests.tldr_summary IS 'One-sentence summary (30-100 chars) - v2.0 design guideline: 50-70 chars ideal';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (17, 'Add length constraints to digest title and tldr_summary fields')
ON CONFLICT (version) DO NOTHING;
