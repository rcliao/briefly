-- Migration: Add TL;DR field to digests
-- Purpose: Store one-line summary for homepage preview

-- Add tldr_summary column to digests table
ALTER TABLE digests ADD COLUMN tldr_summary TEXT;

-- Add comment for documentation
COMMENT ON COLUMN digests.tldr_summary IS 'One-line summary for homepage preview (max 150 chars)';
