-- Schema for Briefly News Aggregator PostgreSQL Database

-- Articles table stores fetched and processed articles
CREATE TABLE IF NOT EXISTS articles (
    id VARCHAR(255) PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    cleaned_text TEXT,
    raw_content TEXT,
    topic_cluster VARCHAR(255),
    cluster_confidence FLOAT,
    embedding JSONB,  -- Store as JSONB for flexibility
    date_fetched TIMESTAMP WITH TIME ZONE NOT NULL,
    date_added TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_content_type CHECK (content_type IN ('html', 'pdf', 'youtube', 'rss'))
);

CREATE INDEX IF NOT EXISTS idx_articles_url ON articles(url);
CREATE INDEX IF NOT EXISTS idx_articles_date_fetched ON articles(date_fetched DESC);
CREATE INDEX IF NOT EXISTS idx_articles_topic_cluster ON articles(topic_cluster);
CREATE INDEX IF NOT EXISTS idx_articles_date_added ON articles(date_added DESC);

-- Summaries table stores LLM-generated summaries
CREATE TABLE IF NOT EXISTS summaries (
    id VARCHAR(255) PRIMARY KEY,
    article_ids JSONB NOT NULL,  -- Array of article IDs
    summary_text TEXT NOT NULL,
    model_used VARCHAR(100) NOT NULL,
    date_created TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_summaries_date_created ON summaries(date_created DESC);
CREATE INDEX IF NOT EXISTS idx_summaries_article_ids ON summaries USING GIN (article_ids);

-- Feeds table stores RSS/Atom feed sources
CREATE TABLE IF NOT EXISTS feeds (
    id VARCHAR(255) PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    last_fetched TIMESTAMP WITH TIME ZONE,
    last_modified TEXT,  -- HTTP Last-Modified header
    etag TEXT,           -- HTTP ETag header
    active BOOLEAN NOT NULL DEFAULT true,
    error_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    date_added TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feeds_active ON feeds(active);
CREATE INDEX IF NOT EXISTS idx_feeds_last_fetched ON feeds(last_fetched);

-- Feed Items table stores individual items from feeds
CREATE TABLE IF NOT EXISTS feed_items (
    id VARCHAR(255) PRIMARY KEY,
    feed_id VARCHAR(255) NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    link TEXT NOT NULL,
    description TEXT,
    published TIMESTAMP WITH TIME ZONE,
    guid TEXT NOT NULL,
    processed BOOLEAN NOT NULL DEFAULT false,
    date_discovered TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(feed_id, guid)
);

CREATE INDEX IF NOT EXISTS idx_feed_items_feed_id ON feed_items(feed_id);
CREATE INDEX IF NOT EXISTS idx_feed_items_processed ON feed_items(processed);
CREATE INDEX IF NOT EXISTS idx_feed_items_published ON feed_items(published DESC);
CREATE INDEX IF NOT EXISTS idx_feed_items_date_discovered ON feed_items(date_discovered DESC);

-- Digests table stores generated digests
CREATE TABLE IF NOT EXISTS digests (
    id VARCHAR(255) PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    content JSONB NOT NULL,  -- Full digest structure as JSON
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_digests_date ON digests(date DESC);
CREATE INDEX IF NOT EXISTS idx_digests_created_at ON digests(created_at DESC);

-- Add comments for documentation
COMMENT ON TABLE articles IS 'Stores fetched articles with embeddings and cluster assignments';
COMMENT ON TABLE summaries IS 'LLM-generated summaries for articles';
COMMENT ON TABLE feeds IS 'RSS/Atom feed sources for news aggregation';
COMMENT ON TABLE feed_items IS 'Individual items discovered in feeds';
COMMENT ON TABLE digests IS 'Generated news digests (daily or weekly)';
