-- Migration: Seed default themes for article classification
-- This provides a starting set of themes covering common tech topics

-- AI & Machine Learning
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-ai-ml',
    'AI & Machine Learning',
    'Articles about artificial intelligence, machine learning models, neural networks, and AI research breakthroughs',
    ARRAY['AI', 'machine learning', 'ML', 'neural networks', 'deep learning', 'LLM', 'GPT', 'transformers', 'NLP', 'computer vision'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Cloud Infrastructure & DevOps
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-cloud-devops',
    'Cloud Infrastructure & DevOps',
    'Cloud services, infrastructure, DevOps practices, Kubernetes, containers, and deployment automation',
    ARRAY['AWS', 'Azure', 'GCP', 'cloud', 'Kubernetes', 'Docker', 'containers', 'DevOps', 'CI/CD', 'infrastructure', 'Terraform', 'IaC'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Software Engineering & Best Practices
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-software-engineering',
    'Software Engineering & Best Practices',
    'Software development methodologies, architecture patterns, testing, code quality, and engineering best practices',
    ARRAY['software engineering', 'architecture', 'design patterns', 'testing', 'TDD', 'clean code', 'refactoring', 'microservices', 'API design'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Web Development & Frontend
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-web-frontend',
    'Web Development & Frontend',
    'Frontend frameworks, web technologies, JavaScript/TypeScript, React, Vue, and modern web development',
    ARRAY['React', 'Vue', 'Angular', 'JavaScript', 'TypeScript', 'frontend', 'web development', 'CSS', 'HTML', 'Next.js', 'Svelte', 'web components'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Data Engineering & Analytics
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-data-engineering',
    'Data Engineering & Analytics',
    'Data pipelines, databases, analytics, data warehousing, ETL, and big data technologies',
    ARRAY['data engineering', 'data pipeline', 'ETL', 'SQL', 'NoSQL', 'PostgreSQL', 'MongoDB', 'data warehouse', 'analytics', 'big data', 'Spark', 'Kafka'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Security & Privacy
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-security',
    'Security & Privacy',
    'Cybersecurity, application security, vulnerability management, encryption, and privacy protection',
    ARRAY['security', 'cybersecurity', 'vulnerability', 'encryption', 'authentication', 'authorization', 'privacy', 'GDPR', 'pentesting', 'zero trust'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Programming Languages & Tools
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-programming-languages',
    'Programming Languages & Tools',
    'Programming language features, updates, tooling, compilers, and language ecosystems',
    ARRAY['Go', 'Python', 'Rust', 'Java', 'C++', 'programming language', 'compiler', 'IDE', 'tooling', 'package manager', 'npm', 'pip'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Mobile Development
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-mobile',
    'Mobile Development',
    'iOS, Android, mobile app development, cross-platform frameworks, and mobile best practices',
    ARRAY['iOS', 'Android', 'mobile', 'Swift', 'Kotlin', 'React Native', 'Flutter', 'mobile app', 'cross-platform'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Open Source & Community
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-open-source',
    'Open Source & Community',
    'Open source projects, community news, licensing, contributions, and ecosystem updates',
    ARRAY['open source', 'OSS', 'GitHub', 'GitLab', 'community', 'contribution', 'license', 'maintainer', 'FOSS'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Product & Startup
INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
VALUES (
    'theme-product-startup',
    'Product & Startup',
    'Product management, startup news, fundraising, growth strategies, and business development',
    ARRAY['startup', 'product management', 'PM', 'fundraising', 'VC', 'growth', 'metrics', 'business', 'MVP', 'product-market fit'],
    true,
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;
