-- Migration: 019_add_tag_system
-- Description: Add tag system for hierarchical article classification
-- Created: 2025-11-11
--
-- Changes:
-- - Create tags table for flat tag taxonomy
-- - Create article_tags junction table for multi-label classification
-- - Seed 50+ tags across 5 top-level themes (GenAI, Technology, Gaming, Healthcare, Finance)
-- - Add indexes for efficient tag-based queries
--
-- Architecture:
-- - 5 user-facing themes (existing themes table) for simple homepage UI
-- - 50+ internal tags (new tags table) for fine-grained clustering
-- - Articles can have multiple tags with relevance scores (0-1 from LLM)
-- - Clustering uses tag similarity within theme groups (hierarchical approach)

-- Create tags table
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    keywords TEXT[], -- Keywords for LLM classification
    theme_id TEXT REFERENCES themes(id), -- Parent theme (optional grouping)
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create article_tags junction table (many-to-many with relevance scores)
CREATE TABLE IF NOT EXISTS article_tags (
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    tag_id TEXT REFERENCES tags(id) ON DELETE CASCADE,
    relevance_score FLOAT CHECK (relevance_score >= 0 AND relevance_score <= 1) DEFAULT 1.0,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (article_id, tag_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_tags_theme_id ON tags(theme_id);
CREATE INDEX IF NOT EXISTS idx_tags_enabled ON tags(enabled) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_article_tags_article_id ON article_tags(article_id);
CREATE INDEX IF NOT EXISTS idx_article_tags_tag_id ON article_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_article_tags_relevance ON article_tags(relevance_score) WHERE relevance_score >= 0.4;

-- Add comments
COMMENT ON TABLE tags IS 'Flat tag taxonomy for fine-grained article classification (internal clustering)';
COMMENT ON TABLE article_tags IS 'Many-to-many relationship between articles and tags with LLM relevance scores';
COMMENT ON COLUMN tags.keywords IS 'Keywords for LLM tag classification (comma-separated in prompt)';
COMMENT ON COLUMN article_tags.relevance_score IS 'LLM confidence score (0-1) from tag classification';

-- ============================================================================
-- SEED TAGS (50+ tags across 5 themes)
-- ============================================================================

-- GENAI THEME TAGS (theme-genai)
INSERT INTO tags (id, name, description, keywords, theme_id) VALUES
('tag-llm', 'Large Language Models', 'LLM training, fine-tuning, inference, model architectures', ARRAY['llm', 'gpt', 'claude', 'language model', 'transformer'], 'theme-genai'),
('tag-rag', 'RAG & Retrieval', 'Retrieval-Augmented Generation, vector search, semantic search', ARRAY['rag', 'retrieval', 'vector search', 'semantic search', 'embeddings'], 'theme-genai'),
('tag-vector-db', 'Vector Databases', 'Vector databases, embeddings storage, similarity search', ARRAY['vector database', 'pinecone', 'chroma', 'pgvector', 'weaviate'], 'theme-genai'),
('tag-fine-tuning', 'Fine-Tuning', 'Model fine-tuning, PEFT, LoRA, instruction tuning', ARRAY['fine-tuning', 'lora', 'peft', 'instruction tuning', 'qlora'], 'theme-genai'),
('tag-prompt-engineering', 'Prompt Engineering', 'Prompting techniques, chain-of-thought, few-shot learning', ARRAY['prompt engineering', 'prompting', 'chain-of-thought', 'few-shot', 'zero-shot'], 'theme-genai'),
('tag-ai-agents', 'AI Agents', 'Autonomous agents, agentic workflows, tool use', ARRAY['ai agents', 'autonomous agents', 'tool use', 'agentic', 'langchain'], 'theme-genai'),
('tag-multimodal', 'Multimodal AI', 'Vision-language models, image generation, speech', ARRAY['multimodal', 'vision', 'image generation', 'dalle', 'stable diffusion'], 'theme-genai'),
('tag-embedding', 'Embeddings', 'Text embeddings, embedding models, semantic similarity', ARRAY['embeddings', 'text embeddings', 'sentence transformers', 'cohere'], 'theme-genai'),
('tag-evaluation', 'AI Evaluation', 'LLM evaluation, benchmarking, EVAL frameworks', ARRAY['evaluation', 'benchmarking', 'eval', 'testing', 'metrics'], 'theme-genai'),
('tag-ai-infra', 'AI Infrastructure', 'ML infrastructure, serving, scaling, orchestration', ARRAY['ml infrastructure', 'mlops', 'ai infrastructure', 'serving', 'deployment'], 'theme-genai'),
('tag-reasoning', 'AI Reasoning', 'Chain-of-thought, reasoning, planning, problem-solving', ARRAY['reasoning', 'planning', 'problem-solving', 'chain-of-thought'], 'theme-genai'),
('tag-ai-safety', 'AI Safety & Alignment', 'AI safety, alignment, RLHF, red teaming', ARRAY['ai safety', 'alignment', 'rlhf', 'red teaming', 'constitutional ai'], 'theme-genai')
ON CONFLICT (id) DO NOTHING;

-- TECHNOLOGY THEME TAGS (theme-technology)
INSERT INTO tags (id, name, description, keywords, theme_id) VALUES
('tag-database', 'Databases', 'Database systems, SQL, NoSQL, performance optimization', ARRAY['database', 'sql', 'postgres', 'mysql', 'mongodb'], 'theme-technology'),
('tag-devops', 'DevOps', 'CI/CD, automation, infrastructure as code, monitoring', ARRAY['devops', 'ci/cd', 'automation', 'jenkins', 'github actions'], 'theme-technology'),
('tag-cloud', 'Cloud Platforms', 'AWS, Azure, GCP, cloud architecture, serverless', ARRAY['cloud', 'aws', 'azure', 'gcp', 'serverless'], 'theme-technology'),
('tag-kubernetes', 'Kubernetes', 'Container orchestration, K8s, deployment, scaling', ARRAY['kubernetes', 'k8s', 'container', 'orchestration', 'helm'], 'theme-technology'),
('tag-security', 'Security', 'Cybersecurity, authentication, encryption, vulnerabilities', ARRAY['security', 'cybersecurity', 'auth', 'encryption', 'vulnerabilities'], 'theme-technology'),
('tag-monitoring', 'Monitoring & Observability', 'Logging, metrics, tracing, APM, alerting', ARRAY['monitoring', 'observability', 'logging', 'metrics', 'tracing'], 'theme-technology'),
('tag-backend', 'Backend Development', 'APIs, microservices, server-side development', ARRAY['backend', 'api', 'microservices', 'rest', 'graphql'], 'theme-technology'),
('tag-frontend', 'Frontend Development', 'React, Vue, Angular, JavaScript, UI/UX', ARRAY['frontend', 'react', 'vue', 'javascript', 'typescript'], 'theme-technology'),
('tag-mobile', 'Mobile Development', 'iOS, Android, React Native, mobile apps', ARRAY['mobile', 'ios', 'android', 'react native', 'flutter'], 'theme-technology'),
('tag-golang', 'Go Programming', 'Golang, Go standard library, concurrency', ARRAY['go', 'golang', 'goroutines', 'concurrency'], 'theme-technology'),
('tag-python', 'Python Programming', 'Python, libraries, frameworks, data science', ARRAY['python', 'django', 'flask', 'pandas', 'numpy'], 'theme-technology'),
('tag-rust', 'Rust Programming', 'Rust, systems programming, memory safety', ARRAY['rust', 'systems programming', 'memory safety', 'cargo'], 'theme-technology'),
('tag-performance', 'Performance Optimization', 'Performance tuning, profiling, benchmarking', ARRAY['performance', 'optimization', 'profiling', 'benchmarking'], 'theme-technology'),
('tag-architecture', 'Software Architecture', 'System design, design patterns, scalability', ARRAY['architecture', 'system design', 'design patterns', 'scalability'], 'theme-technology'),
('tag-networking', 'Networking', 'Network protocols, TCP/IP, HTTP, DNS', ARRAY['networking', 'tcp/ip', 'http', 'dns', 'protocols'], 'theme-technology')
ON CONFLICT (id) DO NOTHING;

-- GAMING THEME TAGS (theme-gaming)
INSERT INTO tags (id, name, description, keywords, theme_id) VALUES
('tag-game-dev', 'Game Development', 'Game engines, game design, development tools', ARRAY['game development', 'game design', 'gamedev', 'game engine'], 'theme-gaming'),
('tag-unity', 'Unity', 'Unity engine, Unity development, C# for games', ARRAY['unity', 'unity3d', 'unity engine'], 'theme-gaming'),
('tag-unreal', 'Unreal Engine', 'Unreal Engine, UE5, C++ for games', ARRAY['unreal', 'unreal engine', 'ue5', 'ue4'], 'theme-gaming'),
('tag-indie', 'Indie Games', 'Independent game development, indie studios', ARRAY['indie', 'indie games', 'independent', 'indie dev'], 'theme-gaming'),
('tag-esports', 'Esports', 'Competitive gaming, esports industry, tournaments', ARRAY['esports', 'competitive gaming', 'tournaments', 'professional gaming'], 'theme-gaming'),
('tag-vr', 'Virtual Reality', 'VR gaming, VR development, immersive experiences', ARRAY['vr', 'virtual reality', 'oculus', 'meta quest'], 'theme-gaming'),
('tag-ar', 'Augmented Reality', 'AR gaming, AR development, mixed reality', ARRAY['ar', 'augmented reality', 'mixed reality', 'arkit'], 'theme-gaming'),
('tag-game-design', 'Game Design', 'Gameplay mechanics, level design, UX for games', ARRAY['game design', 'gameplay', 'level design', 'game mechanics'], 'theme-gaming'),
('tag-game-graphics', 'Game Graphics', 'Rendering, shaders, graphics programming', ARRAY['game graphics', 'rendering', 'shaders', 'graphics programming'], 'theme-gaming')
ON CONFLICT (id) DO NOTHING;

-- HEALTHCARE THEME TAGS (theme-healthcare)
INSERT INTO tags (id, name, description, keywords, theme_id) VALUES
('tag-health-tech', 'Health Technology', 'Healthcare technology, digital health, health IT', ARRAY['health tech', 'healthcare technology', 'digital health', 'health it'], 'theme-healthcare'),
('tag-telemedicine', 'Telemedicine', 'Remote healthcare, telehealth, virtual care', ARRAY['telemedicine', 'telehealth', 'remote healthcare', 'virtual care'], 'theme-healthcare'),
('tag-medical-ai', 'Medical AI', 'AI in healthcare, diagnostics, medical imaging', ARRAY['medical ai', 'healthcare ai', 'medical imaging', 'diagnostics'], 'theme-healthcare'),
('tag-biotech', 'Biotechnology', 'Biotech, genomics, drug discovery, life sciences', ARRAY['biotech', 'biotechnology', 'genomics', 'drug discovery'], 'theme-healthcare'),
('tag-wearables', 'Health Wearables', 'Wearable devices, fitness trackers, health monitoring', ARRAY['wearables', 'fitness trackers', 'health monitoring', 'smartwatch'], 'theme-healthcare'),
('tag-ehr', 'Electronic Health Records', 'EHR systems, medical records, health data', ARRAY['ehr', 'electronic health records', 'medical records', 'health data'], 'theme-healthcare')
ON CONFLICT (id) DO NOTHING;

-- FINANCE THEME TAGS (theme-business)
INSERT INTO tags (id, name, description, keywords, theme_id) VALUES
('tag-fintech', 'Financial Technology', 'Fintech, digital banking, financial services', ARRAY['fintech', 'financial technology', 'digital banking', 'neobank'], 'theme-business'),
('tag-crypto', 'Cryptocurrency', 'Crypto, digital assets, DeFi, web3', ARRAY['crypto', 'cryptocurrency', 'bitcoin', 'ethereum', 'defi'], 'theme-business'),
('tag-blockchain', 'Blockchain', 'Blockchain technology, distributed ledger, smart contracts', ARRAY['blockchain', 'distributed ledger', 'smart contracts', 'web3'], 'theme-business'),
('tag-trading', 'Trading & Markets', 'Stock trading, algorithms, market analysis', ARRAY['trading', 'stock market', 'algorithmic trading', 'quant'], 'theme-business'),
('tag-payments', 'Payments', 'Payment processing, digital wallets, payment tech', ARRAY['payments', 'payment processing', 'digital wallet', 'stripe'], 'theme-business'),
('tag-investing', 'Investing', 'Investment strategies, portfolio management, wealth tech', ARRAY['investing', 'investment', 'portfolio', 'wealth management'], 'theme-business'),
('tag-startup', 'Startups & Business', 'Startup ecosystem, fundraising, entrepreneurship', ARRAY['startup', 'startups', 'entrepreneurship', 'funding', 'venture capital'], 'theme-business'),
('tag-saas', 'SaaS', 'Software as a Service, SaaS business models', ARRAY['saas', 'software as a service', 'b2b', 'enterprise software'], 'theme-business')
ON CONFLICT (id) DO NOTHING;

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (19, 'Add tag system for hierarchical article classification')
ON CONFLICT (version) DO NOTHING;

-- ============================================================================
-- USAGE EXAMPLES:
-- ============================================================================
--
-- 1. Assign tags to article with relevance scores:
--    INSERT INTO article_tags (article_id, tag_id, relevance_score)
--    VALUES
--      ('article-123', 'tag-llm', 0.95),
--      ('article-123', 'tag-rag', 0.87),
--      ('article-123', 'tag-prompt-engineering', 0.72);
--
-- 2. Get all tags for an article:
--    SELECT t.name, at.relevance_score
--    FROM article_tags at
--    JOIN tags t ON at.tag_id = t.id
--    WHERE at.article_id = 'article-123'
--    ORDER BY at.relevance_score DESC;
--
-- 3. Find articles with specific tag:
--    SELECT a.id, a.title, at.relevance_score
--    FROM articles a
--    JOIN article_tags at ON a.id = at.article_id
--    WHERE at.tag_id = 'tag-llm'
--      AND at.relevance_score >= 0.4
--    ORDER BY at.relevance_score DESC;
--
-- 4. Get tag distribution by theme:
--    SELECT th.name AS theme, COUNT(DISTINCT at.article_id) AS article_count
--    FROM tags t
--    JOIN themes th ON t.theme_id = th.id
--    JOIN article_tags at ON t.id = at.tag_id
--    WHERE at.relevance_score >= 0.4
--    GROUP BY th.name
--    ORDER BY article_count DESC;
--
-- 5. Find articles matching multiple tags (tag similarity):
--    SELECT a.id, a.title, COUNT(*) AS matching_tags, AVG(at.relevance_score) AS avg_score
--    FROM articles a
--    JOIN article_tags at ON a.id = at.article_id
--    WHERE at.tag_id IN ('tag-llm', 'tag-rag', 'tag-vector-db')
--      AND at.relevance_score >= 0.4
--    GROUP BY a.id, a.title
--    HAVING COUNT(*) >= 2  -- At least 2 matching tags
--    ORDER BY matching_tags DESC, avg_score DESC;
--
-- ============================================================================
