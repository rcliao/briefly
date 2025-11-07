-- Migration: Replace with focused, mutually exclusive themes
-- Disables all existing themes and creates 5 broad categories

-- Disable all existing themes
UPDATE themes SET enabled = false WHERE enabled = true;

-- Theme 1: Generative AI & Machine Learning
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-genai',
    'GenAI',
    'Large language models, AI research, development tools, applications, and the entire GenAI ecosystem',
    ARRAY[
        -- LLMs & Models
        'LLM', 'large language model', 'GPT', 'Claude', 'Gemini', 'Llama', 'Mistral', 'transformers',
        'attention mechanism', 'neural networks', 'deep learning', 'foundation models',
        -- AI Tools & Platforms
        'ChatGPT', 'OpenAI', 'Anthropic', 'Google AI', 'AI API', 'embeddings', 'vector database',
        'LangChain', 'LlamaIndex', 'AI SDK', 'prompt engineering',
        -- Applications
        'AI agent', 'autonomous agent', 'code generation', 'copilot', 'AI assistant', 'chatbot',
        'multi-agent', 'AI automation', 'workflow AI',
        -- Research & Concepts
        'RLHF', 'fine-tuning', 'model training', 'AI research', 'benchmark', 'MMLU', 'evaluation',
        'machine learning', 'ML', 'artificial intelligence', 'AI', 'computer vision', 'NLP'
    ],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Theme 2: Gaming & Entertainment
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-gaming',
    'Gaming',
    'Video games, game development, esports, streaming, and entertainment technology',
    ARRAY[
        -- Gaming
        'video games', 'gaming', 'game development', 'game engine', 'Unity', 'Unreal Engine',
        'Godot', 'indie games', 'AAA games', 'mobile games', 'game design', 'gameplay',
        -- Platforms
        'PlayStation', 'Xbox', 'Nintendo', 'Steam', 'Epic Games', 'console', 'PC gaming',
        'cloud gaming', 'game streaming', 'GeForce Now', 'xCloud',
        -- Esports & Streaming
        'esports', 'competitive gaming', 'Twitch', 'YouTube Gaming', 'streaming',
        'content creation', 'streamer', 'tournament',
        -- Tech
        'graphics', 'GPU', 'RTX', 'ray tracing', 'VR', 'virtual reality', 'AR', 'metaverse',
        'game AI', 'procedural generation', 'game physics'
    ],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Theme 3: Technology & Innovation
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-technology',
    'Technology',
    'Software development, cloud infrastructure, programming, web development, and general technology news',
    ARRAY[
        -- Software Development
        'software engineering', 'programming', 'coding', 'development', 'software architecture',
        'design patterns', 'clean code', 'refactoring', 'testing', 'TDD', 'API design', 'microservices',
        -- Languages & Tools
        'JavaScript', 'TypeScript', 'Python', 'Go', 'Rust', 'Java', 'C++', 'Kotlin', 'Swift',
        'React', 'Vue', 'Angular', 'Next.js', 'Node.js', 'frontend', 'backend', 'full stack',
        -- Cloud & Infrastructure
        'cloud', 'AWS', 'Azure', 'GCP', 'Kubernetes', 'Docker', 'containers', 'DevOps',
        'CI/CD', 'infrastructure', 'Terraform', 'serverless', 'edge computing',
        -- Data & Databases
        'database', 'SQL', 'PostgreSQL', 'MongoDB', 'NoSQL', 'data engineering', 'ETL',
        'data pipeline', 'analytics', 'big data',
        -- Security & Privacy
        'cybersecurity', 'security', 'privacy', 'encryption', 'authentication', 'vulnerability',
        'zero trust', 'pentesting', 'GDPR',
        -- Tech Industry
        'startup', 'venture capital', 'VC', 'product', 'SaaS', 'open source', 'GitHub',
        'tech news', 'innovation', 'digital transformation'
    ],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Theme 4: Healthcare & Biotech
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-healthcare',
    'Healthcare',
    'Medical technology, biotechnology, digital health, pharmaceuticals, and healthcare innovation',
    ARRAY[
        -- Digital Health
        'digital health', 'health tech', 'medical tech', 'telemedicine', 'telehealth',
        'remote patient monitoring', 'wearables', 'health app', 'fitness tracker',
        -- Medical Technology
        'medical devices', 'diagnostics', 'imaging', 'radiology', 'surgery tech', 'robotics surgery',
        'prosthetics', 'medical AI', 'clinical decision support',
        -- Biotech & Pharma
        'biotechnology', 'biotech', 'pharmaceuticals', 'drug development', 'drug discovery',
        'clinical trials', 'FDA approval', 'gene therapy', 'CRISPR', 'genomics', 'precision medicine',
        -- Healthcare Systems
        'EHR', 'electronic health records', 'hospital', 'healthcare', 'patient care',
        'healthcare IT', 'medical records', 'HIPAA', 'health data',
        -- Research & Innovation
        'medical research', 'life sciences', 'bioinformatics', 'protein folding', 'vaccine',
        'immunotherapy', 'cancer research', 'neuroscience', 'mental health tech'
    ],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Theme 5: Business & Finance
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-business',
    'Finance',
    'Business strategy, fintech, cryptocurrency, economics, and financial technology',
    ARRAY[
        -- Fintech & Payments
        'fintech', 'financial technology', 'payments', 'digital payments', 'mobile payments',
        'payment processing', 'Stripe', 'PayPal', 'banking', 'neobank', 'digital banking',
        -- Cryptocurrency & Blockchain
        'cryptocurrency', 'crypto', 'Bitcoin', 'Ethereum', 'blockchain', 'DeFi',
        'decentralized finance', 'NFT', 'Web3', 'smart contracts', 'digital assets',
        -- Business & Economics
        'business', 'economics', 'finance', 'investment', 'trading', 'stock market',
        'IPO', 'mergers', 'acquisitions', 'M&A', 'earnings', 'revenue', 'profit',
        -- Enterprise
        'enterprise', 'B2B', 'SaaS', 'CRM', 'ERP', 'business software', 'productivity',
        'collaboration', 'project management', 'workflow',
        -- Markets & Regulation
        'SEC', 'regulation', 'compliance', 'financial regulation', 'monetary policy',
        'central bank', 'interest rates', 'inflation', 'market analysis'
    ],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();
