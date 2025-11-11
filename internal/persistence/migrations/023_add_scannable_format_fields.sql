-- Migration: Add v3.0 scannable format fields to digests table
-- Description: Adds columns for TopDevelopments, ByTheNumbers, and WhyItMatters
--              to support the new scannable bullet format introduced in v3.1

-- Add top_developments column (array of bullet point strings)
ALTER TABLE digests
ADD COLUMN IF NOT EXISTS top_developments TEXT[];

-- Add by_the_numbers column (array of stat objects: {stat, context})
ALTER TABLE digests
ADD COLUMN IF NOT EXISTS by_the_numbers JSONB;

-- Add why_it_matters column (single sentence executive summary)
ALTER TABLE digests
ADD COLUMN IF NOT EXISTS why_it_matters VARCHAR(500);

-- Add indexes for querying
CREATE INDEX IF NOT EXISTS idx_digests_top_developments ON digests USING GIN (top_developments);
CREATE INDEX IF NOT EXISTS idx_digests_by_the_numbers ON digests USING GIN (by_the_numbers);

-- Add check constraint for why_it_matters length (20-500 chars for one sentence)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_why_it_matters_length'
    ) THEN
        ALTER TABLE digests
        ADD CONSTRAINT chk_why_it_matters_length
        CHECK (why_it_matters IS NULL OR (length(why_it_matters) >= 20 AND length(why_it_matters) <= 500));
    END IF;
END $$;

COMMENT ON COLUMN digests.top_developments IS 'v3.0: Array of 3-5 bullet points highlighting key developments';
COMMENT ON COLUMN digests.by_the_numbers IS 'v3.0: Array of statistics with format [{stat: "60%", context: "description [1][2]"}]';
COMMENT ON COLUMN digests.why_it_matters IS 'v3.0: Single sentence (20-30 words) executive summary with specific company names';
