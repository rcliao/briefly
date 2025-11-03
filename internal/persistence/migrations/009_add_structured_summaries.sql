-- Migration: 009_add_structured_summaries
-- Description: Add structured summary support (Phase 1)
-- Created: 2025-11-03
-- Purpose: Enable AI-generated summaries with structured sections:
--          - Key Points (3-5 bullets)
--          - Context (background/why it matters)
--          - Main Insight (core takeaway)
--          - Technical Details (optional)
--          - Impact (who/how it affects)

-- Add summary_type column to distinguish between simple and structured summaries
ALTER TABLE summaries
ADD COLUMN IF NOT EXISTS summary_type VARCHAR(20) NOT NULL DEFAULT 'simple';

-- Add structured_content JSONB column for structured summary data
ALTER TABLE summaries
ADD COLUMN IF NOT EXISTS structured_content JSONB;

-- Add constraint to validate summary_type
ALTER TABLE summaries
ADD CONSTRAINT valid_summary_type CHECK (summary_type IN ('simple', 'structured'));

-- Create index on summary_type for filtering
CREATE INDEX IF NOT EXISTS idx_summaries_summary_type ON summaries(summary_type);

-- Create GIN index on structured_content for efficient JSONB queries
CREATE INDEX IF NOT EXISTS idx_summaries_structured_content ON summaries USING GIN (structured_content);

-- Update table comment
COMMENT ON TABLE summaries IS 'LLM-generated summaries for articles. Supports both simple text and structured formats (Phase 1)';
COMMENT ON COLUMN summaries.summary_type IS 'Type of summary: simple (plain text) or structured (with sections)';
COMMENT ON COLUMN summaries.structured_content IS 'JSONB structure: {key_points: [string], context: string, main_insight: string, technical_details: string, impact: string}';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (9, 'Add structured summary support (Phase 1)')
ON CONFLICT (version) DO NOTHING;
