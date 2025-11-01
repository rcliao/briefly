-- Migration: Create "Manual Submissions" feed
-- Description: Creates a special feed for manually submitted URLs
-- Version: 7

-- Create the "Manual Submissions" feed with a fixed ID
INSERT INTO feeds (
    id,
    url,
    title,
    description,
    last_fetched,
    last_modified,
    etag,
    active,
    error_count,
    last_error,
    date_added
) VALUES (
    'manual',  -- Fixed ID for easy reference in code
    'internal://manual',  -- Special URL to indicate it's not a real feed
    'Manual Submissions',
    'Manually submitted URLs for processing',
    NULL,  -- Never fetched
    NULL,  -- No last_modified
    NULL,  -- No etag
    false,  -- Inactive - don't fetch this as a real feed
    0,      -- No errors
    NULL,   -- No last_error
    NOW()   -- date_added
) ON CONFLICT (id) DO UPDATE SET active = false;  -- Update if exists to ensure it's inactive
