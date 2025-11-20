-- Migration 022: Remap old theme IDs to new focused themes
-- Migrates articles, digests, and tags from old theme associations to new single-word themes
--
-- Order of operations (IMPORTANT):
-- 1. Remap articles to new theme IDs
-- 2. Remap digests to new theme IDs
-- 3. Remap tags to new theme IDs (MUST happen before deleting old themes!)
-- 4. Delete old themes (only after all foreign key references are updated)
-- 5. Verify final state

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
-- PART 3: Remap tags to new theme IDs
-- ==============================================================================

-- Remap AI-related tags to 'GenAI'
UPDATE tags
SET theme_id = 'theme-genai'
WHERE theme_id IN (
    'theme-ai-applications',
    'theme-ai-dev-tools',
    'theme-llm-research',
    'theme-ai-ml'
);

-- Remap gaming tags to 'Gaming'
UPDATE tags
SET theme_id = 'theme-gaming'
WHERE theme_id IN (
    'theme-gaming',
    'theme-video-games',
    'theme-esports'
);

-- Remap tech tags to 'Technology'
UPDATE tags
SET theme_id = 'theme-technology'
WHERE theme_id IN (
    'theme-software-engineering',
    'theme-cloud-devops',
    'theme-web-development',
    'theme-web-frontend',
    'theme-programming',
    'theme-programming-languages',
    'theme-open-source',
    'theme-cybersecurity',
    'theme-security'
);

-- Remap healthcare/biotech tags to 'Healthcare'
UPDATE tags
SET theme_id = 'theme-healthcare'
WHERE theme_id IN (
    'theme-healthcare',
    'theme-biotech',
    'theme-medical-tech'
);

-- Remap business/finance tags to 'Finance'
UPDATE tags
SET theme_id = 'theme-business'
WHERE theme_id IN (
    'theme-fintech',
    'theme-cryptocurrency',
    'theme-business',
    'theme-startup',
    'theme-product-startup'
);

-- Remap mobile tags to 'Technology'
UPDATE tags
SET theme_id = 'theme-technology'
WHERE theme_id = 'theme-mobile';

-- Remap data engineering tags to 'Technology'
UPDATE tags
SET theme_id = 'theme-technology'
WHERE theme_id = 'theme-data-engineering';

-- ==============================================================================
-- PART 4: Clean up old themes
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
-- PART 5: Verification
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
