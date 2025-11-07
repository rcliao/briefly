-- Migration 022: Remap old theme IDs to new focused themes
-- Migrates articles and digests from old theme associations to new single-word themes

-- ==============================================================================
-- PART 1: Remap article themes
-- ==============================================================================

-- Remap AI-related themes to 'GenAI'
UPDATE articles
SET theme_id = 'theme-genai'
WHERE theme_id IN (
    'theme-ai-applications',
    'theme-ai-dev-tools',
    'theme-llm-research',
    'theme-ai-ml'
);

-- Remap gaming themes to 'Gaming'
UPDATE articles
SET theme_id = 'theme-gaming'
WHERE theme_id IN (
    'theme-gaming',
    'theme-video-games',
    'theme-esports'
);

-- Remap tech themes to 'Technology'
UPDATE articles
SET theme_id = 'theme-technology'
WHERE theme_id IN (
    'theme-software-engineering',
    'theme-cloud-devops',
    'theme-web-development',
    'theme-programming',
    'theme-open-source',
    'theme-cybersecurity'
);

-- Remap healthcare/biotech themes to 'Healthcare'
UPDATE articles
SET theme_id = 'theme-healthcare'
WHERE theme_id IN (
    'theme-healthcare',
    'theme-biotech',
    'theme-medical-tech'
);

-- Remap business/finance themes to 'Finance'
UPDATE articles
SET theme_id = 'theme-business'
WHERE theme_id IN (
    'theme-fintech',
    'theme-cryptocurrency',
    'theme-business',
    'theme-startup'
);

-- ==============================================================================
-- PART 2: Remap digest_themes relationships
-- ==============================================================================

-- Remap AI-related digest themes to 'GenAI'
UPDATE digest_themes
SET theme_id = 'theme-genai'
WHERE theme_id IN (
    'theme-ai-applications',
    'theme-ai-dev-tools',
    'theme-llm-research',
    'theme-ai-ml'
);

-- Remap gaming digest themes to 'Gaming'
UPDATE digest_themes
SET theme_id = 'theme-gaming'
WHERE theme_id IN ('theme-gaming', 'theme-video-games', 'theme-esports');

-- Remap tech digest themes to 'Technology'
UPDATE digest_themes
SET theme_id = 'theme-technology'
WHERE theme_id IN (
    'theme-software-engineering',
    'theme-cloud-devops',
    'theme-web-development',
    'theme-programming',
    'theme-open-source',
    'theme-cybersecurity'
);

-- Remap healthcare digest themes to 'Healthcare'
UPDATE digest_themes
SET theme_id = 'theme-healthcare'
WHERE theme_id IN ('theme-healthcare', 'theme-biotech', 'theme-medical-tech');

-- Remap business digest themes to 'Finance'
UPDATE digest_themes
SET theme_id = 'theme-business'
WHERE theme_id IN ('theme-fintech', 'theme-cryptocurrency', 'theme-business', 'theme-startup');

-- ==============================================================================
-- PART 3: Clean up old themes
-- ==============================================================================

-- Delete old disabled themes (keep only the 5 new ones)
DELETE FROM themes
WHERE id NOT IN (
    'theme-genai',
    'theme-gaming',
    'theme-technology',
    'theme-healthcare',
    'theme-business'
);

-- ==============================================================================
-- PART 4: Verification
-- ==============================================================================

-- Verify article mapping
SELECT 'Articles by theme:' as summary;
SELECT theme_id, COUNT(*) as article_count
FROM articles
WHERE theme_id IS NOT NULL
GROUP BY theme_id
ORDER BY article_count DESC;

-- Verify digest mapping
SELECT 'Digests by theme:' as summary;
SELECT theme_id, COUNT(*) as digest_count
FROM digest_themes
GROUP BY theme_id
ORDER BY digest_count DESC;

-- Verify only 5 themes remain
SELECT 'Total themes:' as summary;
SELECT COUNT(*) as theme_count FROM themes;
