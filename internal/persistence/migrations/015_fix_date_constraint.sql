-- Migration: 015_fix_date_constraint
-- Description: Ensure legacy date column doesn't block multiple digests per day
-- Created: 2025-11-06
--
-- Changes:
-- - Drop UNIQUE constraint on date column if it still exists
-- - Make date column nullable (optional for v2.0 digests)
-- - Set default value for date to processed_date for new inserts

-- Drop UNIQUE constraint on date column (allow multiple digests per day)
-- Try multiple possible constraint names
ALTER TABLE digests DROP CONSTRAINT IF EXISTS digests_date_key;
ALTER TABLE digests DROP CONSTRAINT IF EXISTS digests_date_unique;

-- Make date column nullable (not all v2.0 fields may populate it)
ALTER TABLE digests ALTER COLUMN date DROP NOT NULL;

-- Add comment explaining legacy status
COMMENT ON COLUMN digests.date IS 'Legacy date column (v1.0) - kept for backward compatibility. Use processed_date for v2.0.';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (15, 'Fix legacy date column constraints for v2.0 compatibility')
ON CONFLICT (version) DO NOTHING;
