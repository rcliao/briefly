# Database Migration Guide

## Overview

Briefly uses a simple but effective migration system to manage PostgreSQL database schema changes. The system:

- ✅ Tracks applied migrations in a `schema_migrations` table
- ✅ Applies migrations sequentially in version order
- ✅ Uses transactions for atomic migrations
- ✅ Embeds migration files in the binary (no external files needed)
- ✅ Supports migration status checking
- ✅ Provides rollback capability (with manual schema reversal)

---

## Quick Start

### 1. Initial Setup

```bash
# Create PostgreSQL database
createdb briefly

# Set connection string
export DATABASE_URL="postgres://user:pass@localhost:5432/briefly?sslmode=disable"

# Or in .briefly.yaml:
database:
  connection_string: "postgres://user:pass@localhost:5432/briefly?sslmode=disable"

# Apply all migrations
./briefly migrate up
```

### 2. Check Migration Status

```bash
./briefly migrate status
```

**Output:**
```
📊 Migration Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Version    Status     Description
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1          ✅ applied  initial schema

Applied: 1 | Pending: 0 | Total: 1
```

---

## Migration Files

### Location

```
internal/persistence/migrations/
├── 001_initial_schema.sql
├── 002_add_user_preferences.sql  (future)
└── 003_add_full_text_search.sql  (future)
```

### Naming Convention

```
<version>_<description>.sql
```

**Examples:**
- `001_initial_schema.sql`
- `002_add_user_preferences.sql`
- `003_add_full_text_search.sql`
- `010_add_search_history.sql`

**Rules:**
- Version must be numeric (001, 002, etc.)
- Versions must be sequential
- Description uses underscores instead of spaces
- Use descriptive names (not `migration_1.sql`)

### Migration File Structure

```sql
-- Migration: 002_add_user_preferences
-- Description: Add user preferences table for customization
-- Created: 2025-10-25

-- Your schema changes here
CREATE TABLE user_preferences (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    key VARCHAR(100) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, key)
);

CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

COMMENT ON TABLE user_preferences IS 'User-specific preferences and settings';

-- Record this migration (IMPORTANT!)
INSERT INTO schema_migrations (version, description)
VALUES (2, 'Add user preferences table')
ON CONFLICT (version) DO NOTHING;
```

---

## Commands

### `briefly migrate up`

Apply all pending migrations.

```bash
./briefly migrate up
```

**What it does:**
1. Creates `schema_migrations` table if it doesn't exist
2. Checks which migrations have been applied
3. Applies pending migrations in version order
4. Records each migration in `schema_migrations`
5. Uses transactions (rollback on failure)

**Output:**
```
{"level":"INFO","msg":"Starting database migration"}
{"level":"INFO","msg":"Found pending migrations","count":1}
{"level":"INFO","msg":"Applying migration","version":1,"description":"initial schema"}
{"level":"INFO","msg":"Successfully applied migration","version":1}
{"level":"INFO","msg":"Migration completed successfully","applied":1}
✅ All migrations applied successfully
```

### `briefly migrate status`

Show migration status (applied vs pending).

```bash
./briefly migrate status
```

**Output:**
```
📊 Migration Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Version    Status     Description
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1          ✅ applied  initial schema
2          ⏳ pending  add user preferences

Applied: 1 | Pending: 1 | Total: 2

Run 'briefly migrate up' to apply pending migrations
```

### `briefly migrate rollback`

Roll back the last migration (⚠️ **USE WITH CAUTION**).

```bash
./briefly migrate rollback
```

**Interactive confirmation:**
```
⚠️  WARNING: Rolling back migrations is dangerous!
This will only remove the migration record from schema_migrations.
You must manually revert any database schema changes.

Are you sure you want to proceed? (yes/no): yes
```

**Force mode (skip confirmation):**
```bash
./briefly migrate rollback --force
```

**What it does:**
1. Removes the last migration record from `schema_migrations`
2. **Does NOT** revert schema changes automatically
3. You must manually write and execute reverse SQL

**Example manual rollback:**
```sql
-- If you rolled back migration 002 (user_preferences)
-- You need to manually run:
DROP TABLE IF EXISTS user_preferences;
```

---

## Schema Migrations Table

The `schema_migrations` table tracks which migrations have been applied:

```sql
CREATE TABLE schema_migrations (
    version INT PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

**Example data:**
```sql
SELECT * FROM schema_migrations ORDER BY version;

 version |        description         |          applied_at
---------|----------------------------|----------------------------
       1 | Initial schema             | 2025-10-24 14:30:00+00
       2 | Add user preferences       | 2025-10-25 09:15:00+00
```

---

## Creating New Migrations

### Step 1: Create Migration File

Create a new file in `internal/persistence/migrations/`:

```bash
# Next version after 001 is 002
vim internal/persistence/migrations/002_add_user_preferences.sql
```

### Step 2: Write Migration SQL

```sql
-- Migration: 002_add_user_preferences
-- Description: Add user preferences for customization
-- Created: 2025-10-25

-- Create new table
CREATE TABLE IF NOT EXISTS user_preferences (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    preference_key VARCHAR(100) NOT NULL,
    preference_value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, preference_key)
);

-- Add indexes
CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

-- Add comments
COMMENT ON TABLE user_preferences IS 'User-specific preferences and settings';

-- IMPORTANT: Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (2, 'Add user preferences table')
ON CONFLICT (version) DO NOTHING;
```

### Step 3: Rebuild Application

The migration files are embedded in the binary, so you must rebuild:

```bash
go build -o briefly ./cmd/briefly
```

### Step 4: Apply Migration

```bash
./briefly migrate status  # Check pending migrations
./briefly migrate up      # Apply them
```

---

## Best Practices

### ✅ DO

1. **Always use transactions** - Wrap multi-statement migrations in transactions
   ```sql
   BEGIN;
   CREATE TABLE ...;
   CREATE INDEX ...;
   COMMIT;
   ```

2. **Use `IF NOT EXISTS`** - Make migrations idempotent when possible
   ```sql
   CREATE TABLE IF NOT EXISTS ...;
   CREATE INDEX IF NOT EXISTS ...;
   ```

3. **Test migrations** - Test on development database first
   ```bash
   # Test database
   DATABASE_URL="postgres://localhost/briefly_test" ./briefly migrate up
   ```

4. **Include rollback notes** - Document how to reverse the migration
   ```sql
   -- Rollback: DROP TABLE user_preferences;
   ```

5. **Sequential versions** - Don't skip version numbers (001, 002, 003...)

6. **Descriptive names** - Use clear descriptions
   - ✅ `002_add_user_preferences.sql`
   - ❌ `migration_2.sql`

### ❌ DON'T

1. **Don't modify applied migrations** - Create a new migration instead
   - ❌ Edit `001_initial_schema.sql`
   - ✅ Create `002_fix_index.sql`

2. **Don't use database-specific syntax** - Stick to PostgreSQL standard
   - ✅ `TIMESTAMP WITH TIME ZONE`
   - ❌ `DATETIME` (MySQL)

3. **Don't forget the INSERT** - Always record the migration
   ```sql
   -- REQUIRED at end of migration:
   INSERT INTO schema_migrations (version, description)
   VALUES (X, 'Description')
   ON CONFLICT (version) DO NOTHING;
   ```

4. **Don't rollback in production** - Plan forward-only migrations

5. **Don't commit broken migrations** - Test before committing

---

## Troubleshooting

### Migration Failed Mid-Way

**Problem:** Migration failed, some changes applied, some not.

**Solution:**
```sql
-- Check what was applied
SELECT * FROM schema_migrations;

-- Manually fix the database
-- Then either:
-- 1. Fix the migration file and retry
-- 2. Mark as applied if you fixed manually:
INSERT INTO schema_migrations (version, description)
VALUES (X, 'Description');
```

### Migration Already Applied

**Problem:** Migration shows as pending but was actually applied.

**Solution:**
```sql
-- Manually insert the migration record
INSERT INTO schema_migrations (version, description)
VALUES (X, 'Your description')
ON CONFLICT (version) DO NOTHING;
```

### Wrong Migration Version

**Problem:** Accidentally created `003` before `002` was applied.

**Solution:**
```bash
# 1. Rename the file to correct version
mv 003_feature.sql 002_feature.sql

# 2. Edit the file to update version in INSERT statement
# Change: VALUES (3, ...) → VALUES (2, ...)

# 3. Rebuild
go build -o briefly ./cmd/briefly
```

### Database Connection Failed

**Problem:** `failed to connect to database: ...`

**Solution:**
```bash
# Check connection string
echo $DATABASE_URL

# Test connection manually
psql "$DATABASE_URL"

# Verify .briefly.yaml
cat .briefly.yaml | grep -A5 database
```

---

## Advanced Usage

### Multiple Environments

```bash
# Development
DATABASE_URL="postgres://localhost/briefly_dev" ./briefly migrate up

# Staging
DATABASE_URL="postgres://staging.example.com/briefly" ./briefly migrate up

# Production
DATABASE_URL="postgres://prod.example.com/briefly" ./briefly migrate up
```

### Migration in CI/CD

```yaml
# .github/workflows/deploy.yml
- name: Run migrations
  run: |
    export DATABASE_URL="${{ secrets.DATABASE_URL }}"
    ./briefly migrate up
```

### Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o briefly ./cmd/briefly

FROM ubuntu:22.04
COPY --from=builder /app/briefly /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/briefly"]

# Run migrations on startup:
# docker run --rm -e DATABASE_URL="..." briefly migrate up
```

### Backup Before Migration

```bash
# Backup production database before migration
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d).sql

# Run migration
./briefly migrate up

# If something goes wrong:
psql $DATABASE_URL < backup_20251024.sql
```

---

## Future Enhancements

Potential improvements to the migration system:

1. **Down Migrations** - Automatic rollback SQL
   ```
   migrations/
   ├── 001_initial_schema.up.sql
   └── 001_initial_schema.down.sql
   ```

2. **Migration Checksums** - Detect modified migrations
   ```sql
   ALTER TABLE schema_migrations ADD COLUMN checksum VARCHAR(64);
   ```

3. **Dry Run Mode** - Show what would be applied
   ```bash
   ./briefly migrate up --dry-run
   ```

4. **Migration Locking** - Prevent concurrent migrations
   ```sql
   SELECT pg_advisory_lock(123456);
   ```

5. **Schema Dump** - Export current schema
   ```bash
   ./briefly migrate dump > schema.sql
   ```

---

## Summary

The migration system provides:

✅ **Simple** - Sequential numbered migrations
✅ **Safe** - Transactional with rollback
✅ **Embedded** - No external files needed
✅ **Tracked** - `schema_migrations` table
✅ **Verifiable** - `migrate status` command

**Key Commands:**
```bash
briefly migrate up       # Apply all pending
briefly migrate status   # Check status
briefly migrate rollback # Undo last (with caution)
```

**File Structure:**
```
internal/persistence/migrations/
└── <version>_<description>.sql
```

For questions or issues, check the troubleshooting section or open an issue on GitHub.
