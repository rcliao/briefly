-- Migration: 004_add_manual_urls
-- Description: Add manual_urls table for user-submitted URLs
-- Created: 2025-10-31

-- Manual URLs table stores user-submitted URLs for processing
CREATE TABLE IF NOT EXISTS manual_urls (
    id VARCHAR(255) PRIMARY KEY,
    url TEXT NOT NULL,
    submitted_by VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'processing', 'processed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_manual_urls_status ON manual_urls(status);
CREATE INDEX IF NOT EXISTS idx_manual_urls_created_at ON manual_urls(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_manual_urls_url ON manual_urls(url);

-- Add table comments
COMMENT ON TABLE manual_urls IS 'User-submitted URLs for manual processing into articles';
COMMENT ON COLUMN manual_urls.status IS 'Processing status: pending, processing, processed, or failed';
COMMENT ON COLUMN manual_urls.submitted_by IS 'User or source that submitted this URL';
COMMENT ON COLUMN manual_urls.error_message IS 'Error details if processing failed';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (4, 'Add manual_urls table for user-submitted URLs')
ON CONFLICT (version) DO NOTHING;
