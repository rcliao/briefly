-- Migration 020: Add quality metrics tracking tables
-- Phase 1: Quality evaluation framework for digests and clustering

-- Digest quality metrics table
-- Tracks comprehensive quality metrics for each generated digest
CREATE TABLE IF NOT EXISTS digest_quality_metrics (
    id SERIAL PRIMARY KEY,
    digest_id VARCHAR(255) NOT NULL REFERENCES digests(id) ON DELETE CASCADE,

    -- Coverage metrics
    article_count INTEGER NOT NULL DEFAULT 0,
    citations_found INTEGER NOT NULL DEFAULT 0,
    coverage_pct NUMERIC(5,4) NOT NULL DEFAULT 0.0, -- 0.0 to 1.0 (100%)
    uncited_articles TEXT, -- JSON array of article numbers

    -- Vagueness detection
    vague_phrases INTEGER NOT NULL DEFAULT 0,
    vague_phrases_list TEXT, -- JSON array of found phrases

    -- Specificity metrics
    word_count INTEGER NOT NULL DEFAULT 0,
    number_count INTEGER NOT NULL DEFAULT 0,
    proper_noun_count INTEGER NOT NULL DEFAULT 0,
    specificity_score INTEGER NOT NULL DEFAULT 0, -- 0-100

    -- Citation analysis
    citation_density NUMERIC(6,2) NOT NULL DEFAULT 0.0, -- Citations per 100 words

    -- Quality assessment
    grade VARCHAR(20) NOT NULL, -- "A - EXCELLENT", "B - GOOD", "C - FAIR", "D - POOR"
    passed BOOLEAN NOT NULL DEFAULT FALSE,
    warnings TEXT, -- JSON array of warning messages

    -- Metadata
    evaluated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    evaluator_version VARCHAR(10) DEFAULT 'v1.0',

    UNIQUE(digest_id) -- One quality metrics record per digest
);

-- Cluster coherence metrics table
-- Tracks clustering quality metrics for each digest generation run
CREATE TABLE IF NOT EXISTS cluster_coherence_metrics (
    id SERIAL PRIMARY KEY,
    digest_id VARCHAR(255) NOT NULL REFERENCES digests(id) ON DELETE CASCADE,

    -- Cluster counts
    num_clusters INTEGER NOT NULL DEFAULT 0,
    num_articles INTEGER NOT NULL DEFAULT 0,
    avg_cluster_size NUMERIC(6,2) NOT NULL DEFAULT 0.0,

    -- Silhouette scores (range: -1 to 1)
    avg_silhouette NUMERIC(6,4) NOT NULL DEFAULT 0.0,
    cluster_silhouettes TEXT, -- JSON array of per-cluster scores

    -- Cohesion metrics (0 to 1)
    avg_intra_cluster_similarity NUMERIC(6,4) NOT NULL DEFAULT 0.0,
    intra_cluster_similarities TEXT, -- JSON array of per-cluster cohesion

    -- Separation metrics (0 to 1)
    avg_inter_cluster_distance NUMERIC(6,4) NOT NULL DEFAULT 0.0,

    -- Quality assessment
    coherence_grade VARCHAR(20) NOT NULL, -- "A - EXCELLENT", "B - GOOD", etc.
    passed BOOLEAN NOT NULL DEFAULT FALSE,
    issues TEXT, -- JSON array of issue messages

    -- Metadata
    evaluated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    clustering_algorithm VARCHAR(50) DEFAULT 'kmeans', -- 'kmeans', 'hdbscan', etc.

    UNIQUE(digest_id) -- One coherence metrics record per digest
);

-- Quality thresholds configuration table
-- Allows storing different quality threshold profiles
CREATE TABLE IF NOT EXISTS quality_thresholds (
    id SERIAL PRIMARY KEY,
    profile_name VARCHAR(50) NOT NULL UNIQUE,

    -- Digest quality thresholds
    min_coverage_pct NUMERIC(4,3) NOT NULL DEFAULT 0.80,
    max_vague_phrases INTEGER NOT NULL DEFAULT 2,
    min_word_count INTEGER NOT NULL DEFAULT 150,
    max_word_count INTEGER NOT NULL DEFAULT 400,
    min_specificity_score INTEGER NOT NULL DEFAULT 50,
    min_citation_density NUMERIC(4,2) NOT NULL DEFAULT 2.0,

    -- Cluster quality thresholds
    min_silhouette_score NUMERIC(4,3) NOT NULL DEFAULT 0.30,
    min_intra_cluster_sim NUMERIC(4,3) NOT NULL DEFAULT 0.50,
    min_inter_cluster_dist NUMERIC(4,3) NOT NULL DEFAULT 0.30,

    -- Grade thresholds (JSON objects)
    grade_a_thresholds TEXT, -- JSON: {min_coverage, max_vague, ...}
    grade_b_thresholds TEXT,
    grade_c_thresholds TEXT,

    -- Metadata
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default quality thresholds profile
INSERT INTO quality_thresholds (
    profile_name,
    min_coverage_pct,
    max_vague_phrases,
    min_word_count,
    max_word_count,
    min_specificity_score,
    min_citation_density,
    min_silhouette_score,
    min_intra_cluster_sim,
    min_inter_cluster_dist,
    grade_a_thresholds,
    grade_b_thresholds,
    grade_c_thresholds,
    is_active
) VALUES (
    'default',
    0.80,  -- 80% article coverage
    2,     -- max 2 vague phrases
    150,   -- min 150 words
    400,   -- max 400 words
    50,    -- min specificity score of 50/100
    2.0,   -- min 2 citations per 100 words
    0.30,  -- min silhouette 0.3
    0.50,  -- min intra-cluster similarity 0.5
    0.30,  -- min inter-cluster distance 0.3
    '{"min_coverage": 0.90, "max_vague": 1, "min_specificity": 70, "require_numbers": true, "require_names": true, "min_silhouette": 0.5}',
    '{"min_coverage": 0.80, "max_vague": 2, "min_specificity": 50, "require_numbers": true, "require_names": false, "min_silhouette": 0.4}',
    '{"min_coverage": 0.60, "max_vague": 3, "min_specificity": 30, "require_numbers": false, "require_names": false, "min_silhouette": 0.3}',
    TRUE
) ON CONFLICT (profile_name) DO NOTHING;

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_digest_quality_metrics_digest_id ON digest_quality_metrics(digest_id);
CREATE INDEX IF NOT EXISTS idx_digest_quality_metrics_grade ON digest_quality_metrics(grade);
CREATE INDEX IF NOT EXISTS idx_digest_quality_metrics_passed ON digest_quality_metrics(passed);
CREATE INDEX IF NOT EXISTS idx_digest_quality_metrics_evaluated_at ON digest_quality_metrics(evaluated_at DESC);

CREATE INDEX IF NOT EXISTS idx_cluster_coherence_metrics_digest_id ON cluster_coherence_metrics(digest_id);
CREATE INDEX IF NOT EXISTS idx_cluster_coherence_metrics_grade ON cluster_coherence_metrics(coherence_grade);
CREATE INDEX IF NOT EXISTS idx_cluster_coherence_metrics_passed ON cluster_coherence_metrics(passed);
CREATE INDEX IF NOT EXISTS idx_cluster_coherence_metrics_evaluated_at ON cluster_coherence_metrics(evaluated_at DESC);

CREATE INDEX IF NOT EXISTS idx_quality_thresholds_active ON quality_thresholds(is_active) WHERE is_active = TRUE;

-- Add comments for documentation
COMMENT ON TABLE digest_quality_metrics IS 'Tracks comprehensive quality metrics for each generated digest including coverage, vagueness, specificity, and citation density';
COMMENT ON TABLE cluster_coherence_metrics IS 'Tracks clustering quality metrics including silhouette scores, intra-cluster cohesion, and inter-cluster separation';
COMMENT ON TABLE quality_thresholds IS 'Stores configurable quality threshold profiles for different quality standards';

COMMENT ON COLUMN digest_quality_metrics.coverage_pct IS 'Percentage of articles cited in digest (0.0-1.0)';
COMMENT ON COLUMN digest_quality_metrics.specificity_score IS 'Overall specificity score (0-100) based on numbers, names, and lack of vague phrases';
COMMENT ON COLUMN digest_quality_metrics.citation_density IS 'Number of article citations per 100 words';
COMMENT ON COLUMN digest_quality_metrics.grade IS 'Letter grade: A (excellent), B (good), C (fair), D (poor)';

COMMENT ON COLUMN cluster_coherence_metrics.avg_silhouette IS 'Average silhouette score across all clusters (-1 to 1, higher is better)';
COMMENT ON COLUMN cluster_coherence_metrics.avg_intra_cluster_similarity IS 'Average cosine similarity within clusters (0-1, higher means better cohesion)';
COMMENT ON COLUMN cluster_coherence_metrics.avg_inter_cluster_distance IS 'Average distance between cluster centroids (0-1, higher means better separation)';
