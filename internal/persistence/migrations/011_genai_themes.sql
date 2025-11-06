-- Migration: Replace default themes with 3 GenAI-focused themes
-- Disables old themes and adds new GenAI-specific themes

-- Disable all existing themes
UPDATE themes SET enabled = false WHERE enabled = true;

-- Theme 1: LLM Models & Research
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-llm-research',
    'LLM Models & Research',
    'Large language model releases, research papers, architecture innovations, benchmarks, and model capabilities',
    ARRAY['LLM', 'large language model', 'GPT', 'Claude', 'Gemini', 'Llama', 'Mistral', 'transformers', 'attention', 'model architecture', 'research paper', 'benchmark', 'MMLU', 'evaluation', 'fine-tuning', 'RLHF', 'prompt engineering'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Theme 2: AI Development Tools & Platforms
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-ai-dev-tools',
    'AI Development Tools & Platforms',
    'AI development platforms, APIs, SDKs, developer tools, and services from OpenAI, Anthropic, Google, and others',
    ARRAY['ChatGPT', 'Claude', 'Gemini', 'OpenAI API', 'Anthropic', 'Google AI', 'API', 'SDK', 'developer tools', 'AI platform', 'model API', 'embeddings', 'vector database', 'LangChain', 'LlamaIndex', 'AI infrastructure', 'deployment', 'serverless AI'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Theme 3: AI Applications & Use Cases
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-ai-applications',
    'AI Applications & Use Cases',
    'Real-world AI applications, agent frameworks, autonomous systems, production deployments, and AI product launches',
    ARRAY['AI agent', 'autonomous agent', 'AI application', 'production AI', 'AI product', 'code generation', 'copilot', 'AI assistant', 'workflow automation', 'AI tools', 'ChatGPT plugins', 'Claude projects', 'AI use case', 'agent framework', 'AutoGPT', 'multi-agent', 'AI deployment', 'AI integration'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    keywords = EXCLUDED.keywords,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();
