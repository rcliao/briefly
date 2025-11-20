# Database Migration Guide

## Overview

This project uses a **migration-based approach** to manage database schema changes. The migration system handles:

- ✅ Schema creation (tables, indexes, constraints)
- ✅ Extension enablement (pgvector, uuid-ossp, etc.)
- ✅ Data migrations and transformations
- ✅ Graceful degradation when optional features unavailable

---

## Quick Start

### Run All Migrations

```bash
./run_migrations.sh
```

This applies all pending migrations in order and shows:
- Migration status (applied/pending)
- Helpful notices (e.g., pgvector installation instructions)
- Final database status

### Apply Single Migration

```bash
source .env
psql "$DATABASE_URL" -f internal/persistence/migrations/002_enable_pgvector.sql
```

---

## Migration: pgvector Extension

### What Migration 002 Does

**Migration**: `002_enable_pgvector.sql`

**Purpose**: Enable the pgvector extension for semantic search

**Behavior**:
1. ✅ **If pgvector installed**: Enables extension, shows success message
2. ⚠️ **If pgvector NOT installed**: Shows installation instructions, continues gracefully

**Important**: This migration **enables** the extension but CANNOT install the system package.

### Understanding: Extension vs System Package

| Level | What | How | Who |
|-------|------|-----|-----|
| **System Package** | pgvector binaries | `brew install pgvector` | System admin |
| **Database Extension** | Enable in database | `CREATE EXTENSION vector;` | Migration |

**Analogy**:
- System package = Installing Python interpreter
- Database extension = `import numpy` in your code

### Installation Options

#### Option 1: Homebrew (macOS)

```bash
# Install pgvector
brew install pgvector

# Restart PostgreSQL
brew services restart postgresql@16  # Adjust version

# Re-run migrations
./run_migrations.sh
```

**Expected Output**:
```
✅ pgvector extension enabled successfully!

Features now available:
  • Vector similarity search (cosine, L2, inner product)
  • HNSW and IVFFlat indexing for fast approximate search
  • Up to 16,000 dimensions per vector
```

#### Option 2: Docker (Recommended for Development)

```bash
# Stop existing database
docker stop postgres  # If running

# Start PostgreSQL with pgvector pre-installed
docker run -d \
  --name postgres-vector \
  -e POSTGRES_PASSWORD=briefly_dev_password \
  -e POSTGRES_USER=briefly \
  -e POSTGRES_DB=briefly \
  -p 5432:5432 \
  pgvector/pgvector:pg16

# Update .env with new DATABASE_URL
echo "DATABASE_URL=postgresql://briefly:briefly_dev_password@localhost:5432/briefly" > .env

# Run migrations
./run_migrations.sh

# Restore data from backup (if needed)
psql $DATABASE_URL < backup.sql
```

#### Option 3: Ubuntu/Debian

```bash
# Add PostgreSQL APT repository (if not already)
sudo apt install postgresql-common
sudo /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh

# Install pgvector
sudo apt install postgresql-16-pgvector

# Restart PostgreSQL
sudo systemctl restart postgresql

# Re-run migrations
./run_migrations.sh
```

#### Option 4: From Source

```bash
# Install dependencies
sudo apt install build-essential postgresql-server-dev-16

# Clone and build
git clone https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install

# Restart PostgreSQL
sudo systemctl restart postgresql

# Re-run migrations
./run_migrations.sh
```

---

## Migration Workflow

### How Migrations Work

1. **Schema Migrations Table**
   - Tracks which migrations have been applied
   - Prevents duplicate application
   - Records timestamp

2. **Sequential Application**
   - Migrations run in numeric order (001, 002, 003, ...)
   - Skips already-applied migrations
   - Stops on first error (unless migration handles it)

3. **Graceful Degradation**
   - Optional features (like pgvector) fail gracefully
   - System continues with fallback behavior
   - Clear error messages guide next steps

### Migration Lifecycle

```
┌─────────────────┐
│  Create .sql    │
│  migration file │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Test locally   │
│  with psql      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Add to repo    │
│  (migrations/)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Run via script │
│  ./run_migra... │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Verify results │
│  (tests pass)   │
└─────────────────┘
```

---

## Migration Files

### Current Migrations

| # | File | Description | Status |
|---|------|-------------|--------|
| 001 | `initial_schema.sql` | Core tables (articles, summaries, feeds, digests) | ✅ Applied |
| 002 | `enable_pgvector.sql` | Enable pgvector extension | ⚠️ Pending install |
| 003 | `add_themes.sql` | Theme classification system | ✅ Applied |
| 004 | `add_manual_urls.sql` | User-submitted URLs | ✅ Applied |
| 005 | `add_article_themes.sql` | Article-theme relationships | ✅ Applied |
| ... | ... | ... | ... |
| 018 | `pgvector_embeddings.sql` | JSONB → VECTOR conversion | ⏸️ Waits for 002 |
| 019 | `add_tag_system.sql` | Tag-based hierarchical clustering | ✅ Applied |

### Key Migrations for Phase 2

**Migration 002: Enable pgvector**
- Enables vector extension
- Shows installation instructions if unavailable
- Required for semantic search

**Migration 018: Convert Embeddings**
- Creates `embedding_vector` VECTOR(768) column
- Migrates JSONB embeddings to vector format
- Creates IVFFlat index for fast search
- **Depends on**: Migration 002 (pgvector enabled)

**Migration 019: Tag System**
- Creates `tags` and `article_tags` tables
- Seeds 50 tags across 5 themes
- Required for tag-aware clustering

---

## Verifying Migration Status

### Check Applied Migrations

```bash
source .env
psql "$DATABASE_URL" -c "SELECT version, description, applied_at FROM schema_migrations ORDER BY version;"
```

**Expected Output**:
```
 version |                   description                    |          applied_at
---------+--------------------------------------------------+-------------------------------
       1 | Initial schema for news aggregator               | 2025-11-13 10:30:00.123456-08
       2 | Enable pgvector extension                        | 2025-11-13 14:15:30.654321-08
       3 | Add themes table                                 | 2025-11-13 10:30:05.234567-08
...
```

### Check pgvector Status

```bash
psql "$DATABASE_URL" -c "SELECT * FROM pg_extension WHERE extname='vector';"
```

**If Installed**:
```
  oid  | extname | extowner | extnamespace | extrelocatable | extversion
-------+---------+----------+--------------+----------------+------------
 16384 | vector  |       10 |         2200 | f              | 0.5.1
```

**If Not Installed**:
```
(0 rows)
```

### Check Embedding Column Type

```bash
psql "$DATABASE_URL" -c "\d articles" | grep embedding
```

**Before Migration 018** (JSONB):
```
 embedding       | jsonb           |
```

**After Migration 018** (VECTOR):
```
 embedding       | jsonb           |           | Deprecated
 embedding_vector| vector(768)     |           | Fast semantic search
```

---

## Troubleshooting

### Migration 002 Shows "pgvector not available"

**Symptom**:
```
⚠️  pgvector extension not available on this PostgreSQL installation
```

**Cause**: System package not installed

**Fix**: Install pgvector at system level (see installation options above)

**Verify**:
```bash
# After installation
./run_migrations.sh

# Should show:
✅ pgvector extension enabled successfully!
```

---

### Migration 018 Skips Vector Conversion

**Symptom**:
```
pgvector not available - skipping vector column and index creation
```

**Cause**: Migration 002 hasn't successfully enabled pgvector

**Fix**:
1. Install pgvector system package
2. Re-run migrations: `./run_migrations.sh`
3. Migration 018 will automatically run conversion

**Verify**:
```bash
psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM articles WHERE embedding_vector IS NOT NULL;"
```

Should show count of articles with vector embeddings.

---

### Duplicate Migration Application

**Symptom**:
```
ERROR: duplicate key value violates unique constraint "schema_migrations_pkey"
```

**Cause**: Migration already applied but trying to apply again

**Solution**: This is normal - migration system tracks what's applied. Safe to ignore.

---

## Best Practices

### Creating New Migrations

1. **Use Sequential Numbering**
   ```
   024_add_new_feature.sql  # Next available number
   ```

2. **Descriptive Names**
   ```
   ✅ 024_add_user_preferences.sql
   ❌ 024_changes.sql
   ```

3. **Include Header Comment**
   ```sql
   -- Migration: 024_add_user_preferences
   -- Description: Add user preferences table for customization
   -- Created: 2025-11-13
   ```

4. **Handle Errors Gracefully**
   ```sql
   DO $$
   BEGIN
       -- Your migration code
   EXCEPTION WHEN OTHERS THEN
       RAISE NOTICE 'Optional feature - gracefully degrading';
   END $$;
   ```

5. **Record Migration**
   ```sql
   INSERT INTO schema_migrations (version, description)
   VALUES (24, 'Add user preferences table')
   ON CONFLICT (version) DO NOTHING;
   ```

6. **Test Before Committing**
   ```bash
   # Test on clean database
   psql test_db -f migrations/024_add_user_preferences.sql

   # Verify idempotency (safe to run twice)
   psql test_db -f migrations/024_add_user_preferences.sql
   ```

---

## Migration Runner Features

### What `run_migrations.sh` Does

1. **Checks Connection** - Verifies DATABASE_URL
2. **Creates Tracking Table** - Ensures schema_migrations exists
3. **Lists Applied** - Shows which migrations already ran
4. **Applies Pending** - Runs missing migrations in order
5. **Shows Notices** - Displays helpful messages (e.g., install instructions)
6. **Reports Status** - Final database state summary

### Output Sections

**Connection Status**:
```
✅ Connected to database
```

**Migration Progress**:
```
🔄 Applying migration 2: enable_pgvector
✅ Migration 2 completed
```

**Helpful Notices**:
```
⚠️  pgvector extension not available
📋 Installation Instructions:
    brew install pgvector
    ...
```

**Final Status**:
```
📊 Database Status:
 version |              description
---------+----------------------------------------
       1 | Initial schema for news aggregator
       2 | Enable pgvector extension
      ...
```

---

## FAQ

### Q: Why not install pgvector via migration?

**A**: Migrations run at the **database level** (SQL). System packages require **OS-level** installation (Homebrew, apt, etc.). PostgreSQL extensions must be installed on the server filesystem before they can be enabled in the database.

### Q: What if I don't install pgvector?

**A**: System works fine! It uses **JSONB embeddings** as a fallback. You'll get:
- ✅ All features work
- ⚠️ Slower semantic search (1-10ms vs <100µs)
- ⚠️ Larger storage (10KB vs 3KB per embedding)

### Q: Can I install pgvector later?

**A**: Yes! Just:
1. Install system package
2. Run `./run_migrations.sh`
3. Migration 018 automatically converts existing embeddings

### Q: How do I rollback a migration?

**A**: Currently, migrations are **forward-only**. To rollback:
1. Restore database from backup
2. Or manually reverse the changes via SQL

(Future: Consider adding rollback scripts for critical migrations)

### Q: What's the difference between migration 002 and 018?

| Migration | Purpose | Dependency |
|-----------|---------|------------|
| **002** | Enable extension | System package installed |
| **018** | Convert data | Migration 002 successful |

002 enables the feature, 018 uses the feature.

---

## Summary

✅ **Migration System Ready**
- Run `./run_migrations.sh` to apply all pending migrations
- pgvector integration handled gracefully
- Clear instructions when system packages needed

✅ **Current Status**
- Migration 002 created and ready
- Shows helpful installation instructions
- System continues working without pgvector

✅ **Next Steps**
1. Install pgvector system package (optional but recommended)
2. Re-run migrations to enable extension
3. Migration 018 will auto-convert embeddings
4. Ready for Phase 2 semantic clustering!
