-- Migration: 003_add_themes
-- Description: Add themes table for article classification and filtering
-- Created: 2025-10-31

-- Themes table stores user-defined themes for content filtering
CREATE TABLE IF NOT EXISTS themes (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    keywords TEXT[], -- PostgreSQL array for comma-separated keywords
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_themes_enabled ON themes(enabled);
CREATE INDEX IF NOT EXISTS idx_themes_name ON themes(name);
CREATE INDEX IF NOT EXISTS idx_themes_created_at ON themes(created_at DESC);

-- Add table comment
COMMENT ON TABLE themes IS 'User-defined themes for article classification and filtering';
COMMENT ON COLUMN themes.keywords IS 'Array of keywords used for theme classification';
COMMENT ON COLUMN themes.enabled IS 'Whether this theme is active for classification';

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (3, 'Add themes table for article classification')
ON CONFLICT (version) DO NOTHING;
