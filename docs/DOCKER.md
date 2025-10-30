# Docker Setup Guide

## Quick Start

### 1. Start PostgreSQL

```bash
# Start PostgreSQL container
docker-compose up -d

# Or use Makefile
make docker-up
```

**Default credentials:**
- Host: `localhost:5432`
- Database: `briefly`
- User: `briefly`
- Password: `briefly_dev_password`

**Connection string:**
```bash
export DATABASE_URL="postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"
```

### 2. Run Migrations

```bash
# Build briefly
make build

# Apply migrations
make migrate

# Check status
make migrate-status
```

### 3. Start Using

```bash
# Add a feed
./briefly feed add https://hnrss.org/newest

# Aggregate news
./briefly aggregate --since 24

# List feeds
./briefly feed list
```

---

## Docker Compose Configurations

### Development (Basic)

**Start PostgreSQL only:**
```bash
docker-compose up -d
```

**Services:**
- PostgreSQL on `localhost:5432`

### Development (Full)

**Start PostgreSQL + pgAdmin:**
```bash
docker-compose --profile dev up -d
# Or
make docker-up-dev
```

**Services:**
- PostgreSQL on `localhost:5432`
- pgAdmin on `http://localhost:5050`
  - Email: `admin@briefly.local`
  - Password: `admin`

### Development (Complete)

**Start PostgreSQL + pgAdmin + Redis:**
```bash
docker-compose --profile full up -d
# Or
make docker-up-full
```

**Services:**
- PostgreSQL on `localhost:5432`
- pgAdmin on `http://localhost:5050`
- Redis on `localhost:6379`

### Production

**Start production PostgreSQL with backups:**
```bash
# Set production password in .env first!
cp .env.example .env
# Edit .env and set POSTGRES_PASSWORD

docker-compose -f docker-compose.prod.yml up -d
# Or
make docker-up-prod
```

**Features:**
- Resource limits (2GB RAM, 2 CPU)
- Automated daily backups (kept for 7 days)
- Production-tuned PostgreSQL config
- Only binds to localhost (not exposed publicly)
- Health checks and restart policies

---

## Configuration Files

### `docker-compose.yml` (Development)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: briefly
      POSTGRES_PASSWORD: briefly_dev_password
      POSTGRES_DB: briefly
    volumes:
      - postgres_data:/var/lib/postgresql/data
```

**Features:**
- PostgreSQL 16 (Alpine for smaller image)
- Persistent data volume
- Health checks
- Optional pgAdmin and Redis (via profiles)

### `docker-compose.prod.yml` (Production)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "127.0.0.1:5432:5432"  # localhost only
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G

  postgres-backup:
    image: prodrigestivill/postgres-backup-local
    environment:
      SCHEDULE: "@daily"
      BACKUP_KEEP_DAYS: 7
```

**Features:**
- Production-tuned PostgreSQL config
- Resource limits
- Automated backups
- Only binds to localhost
- Logging with rotation

---

## Environment Variables

### `.env` File

```bash
# Copy example
cp .env.example .env

# Edit .env
vim .env
```

**Required variables:**
```bash
# Database
DATABASE_URL=postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable
POSTGRES_USER=briefly
POSTGRES_PASSWORD=briefly_dev_password  # Change in production!
POSTGRES_DB=briefly

# AI
GEMINI_API_KEY=your-key-here
```

**Optional variables:**
```bash
# OpenAI (for future banner generation)
OPENAI_API_KEY=your-key-here

# Logging
LOG_LEVEL=info
DEBUG=false

# Cache
CACHE_DIR=.briefly-cache
CACHE_TTL_HOURS=168
```

---

## Makefile Commands

### Docker Management

```bash
# Start services
make docker-up           # PostgreSQL only
make docker-up-dev       # + pgAdmin
make docker-up-full      # + pgAdmin + Redis
make docker-up-prod      # Production mode

# Stop services
make docker-down

# View logs
make docker-logs

# Clean volumes (⚠️ deletes data!)
make docker-clean
```

### Database Management

```bash
# Run migrations
make migrate

# Check migration status
make migrate-status

# Open PostgreSQL shell
make db-shell

# Reset database (⚠️ deletes data!)
make db-reset

# Backup database
make backup
```

### Application

```bash
# Build
make build

# Test
make test

# Add feed
make feed-add URL=https://example.com/feed.xml

# List feeds
make feed-list

# View feed statistics
make feed-stats

# Aggregate news
make aggregate
```

### First-Time Setup

```bash
# One command to set everything up
make setup

# Equivalent to:
# - docker-compose up -d
# - sleep 5
# - make migrate
```

---

## PostgreSQL Configuration

### Development Config

**Location:** `docker/postgres/postgresql.conf`

**Key settings:**
```
max_connections = 100
shared_buffers = 256MB
log_statement = 'all'  # Log all queries
log_duration = on
```

**Features:**
- Verbose logging for debugging
- Optimized for local development
- Logs all queries

### Production Config

**Location:** `docker/postgres/postgresql.prod.conf`

**Key settings:**
```
max_connections = 200
shared_buffers = 1GB
log_min_duration_statement = 1000  # Only slow queries
autovacuum_naptime = 30s
```

**Features:**
- Tuned for 4GB RAM server
- Logs only slow queries (>1s)
- Aggressive autovacuum
- Better connection pooling

---

## pgAdmin Setup

### Access pgAdmin

1. Start with dev profile:
   ```bash
   make docker-up-dev
   ```

2. Open http://localhost:5050

3. Login:
   - Email: `admin@briefly.local`
   - Password: `admin`

### Connect to Database

1. Right-click "Servers" → "Register" → "Server"

2. **General tab:**
   - Name: `Briefly Local`

3. **Connection tab:**
   - Host: `postgres` (Docker network name)
   - Port: `5432`
   - Database: `briefly`
   - Username: `briefly`
   - Password: `briefly_dev_password`

4. Click "Save"

---

## Backup & Restore

### Automated Backups (Production)

```bash
# Production setup includes automatic daily backups
docker-compose -f docker-compose.prod.yml up -d

# Backups saved in: ./backups/
ls -lh backups/
```

**Retention policy:**
- Daily backups: 7 days
- Weekly backups: 4 weeks
- Monthly backups: 6 months

### Manual Backup

```bash
# Using Makefile
make backup

# Or manually
docker-compose exec -T postgres pg_dump -U briefly -d briefly > backup_$(date +%Y%m%d).sql
```

### Restore from Backup

```bash
# Stop containers
make docker-down

# Clean volumes
make docker-clean

# Start fresh
make docker-up

# Wait for PostgreSQL
sleep 5

# Restore backup
cat backup_20251024.sql | docker-compose exec -T postgres psql -U briefly -d briefly

# Run migrations (if schema changed)
make migrate
```

---

## Troubleshooting

### Container Won't Start

**Check logs:**
```bash
docker-compose logs postgres
```

**Common issues:**
- Port 5432 already in use
  ```bash
  # Check what's using port 5432
  lsof -i :5432

  # Stop local PostgreSQL
  brew services stop postgresql
  ```

- Volume permission issues
  ```bash
  # Reset volumes
  make docker-clean
  make docker-up
  ```

### Can't Connect to Database

**Check container status:**
```bash
docker-compose ps
```

**Check health:**
```bash
docker-compose exec postgres pg_isready -U briefly
```

**Test connection:**
```bash
docker-compose exec postgres psql -U briefly -d briefly -c "SELECT version();"
```

**Common fixes:**
- Wait for PostgreSQL to start (can take 5-10 seconds)
- Check `DATABASE_URL` environment variable
- Verify credentials in `.env`

### Migration Fails

**Check migration status:**
```bash
make migrate-status
```

**Check logs:**
```bash
./briefly migrate up 2>&1 | tee migration.log
```

**Manual fix:**
```bash
# Open PostgreSQL shell
make db-shell

# Check migrations table
SELECT * FROM schema_migrations;

# If needed, manually mark as applied
INSERT INTO schema_migrations (version, description)
VALUES (1, 'Initial schema')
ON CONFLICT DO NOTHING;
```

### Database Reset

**Nuclear option (⚠️ deletes everything):**
```bash
make db-reset
```

**Step by step:**
```bash
# 1. Stop containers
make docker-down

# 2. Remove volumes
docker volume rm briefly_postgres_data

# 3. Start fresh
make docker-up

# 4. Run migrations
make migrate
```

---

## Resource Management

### Check Resource Usage

```bash
# Container stats
docker stats briefly-postgres

# Disk usage
docker system df

# Volume size
docker volume ls
docker volume inspect briefly_postgres_data | grep Mountpoint
```

### Cleanup

```bash
# Remove unused images
docker image prune -a

# Remove unused volumes
docker volume prune

# Full cleanup (⚠️ affects all Docker resources)
docker system prune -a --volumes
```

---

## Production Deployment

### Cloud Providers

**Render.com:**
```bash
# Use managed PostgreSQL
# Set DATABASE_URL in environment variables
# Deploy briefly as a background worker
```

**Railway.app:**
```yaml
# railway.json
{
  "build": {
    "builder": "DOCKERFILE"
  },
  "deploy": {
    "numReplicas": 1,
    "sleepApplication": false,
    "cronSchedule": "0 6 * * *"  # Daily at 6am
  }
}
```

**Fly.io:**
```bash
# Create Postgres database
fly postgres create

# Attach to app
fly postgres attach my-briefly-db

# Deploy
fly deploy
```

### Self-Hosted (VPS)

```bash
# 1. Install Docker & Docker Compose
curl -fsSL https://get.docker.com | sh

# 2. Clone repo
git clone https://github.com/you/briefly.git
cd briefly

# 3. Configure
cp .env.example .env
vim .env  # Set POSTGRES_PASSWORD and GEMINI_API_KEY

# 4. Start production
make docker-up-prod

# 5. Run migrations
make migrate

# 6. Add cron job
crontab -e
# Add: 0 6 * * * cd /path/to/briefly && ./briefly aggregate --since 24
```

---

## Security Best Practices

### Development

- ✅ Use `.env` file (gitignored)
- ✅ Never commit passwords
- ✅ Use default passwords (not critical for local)

### Production

- ✅ Use strong passwords (20+ chars)
- ✅ Use environment variables
- ✅ Bind to localhost only
- ✅ Enable SSL for PostgreSQL
- ✅ Regular backups
- ✅ Resource limits
- ✅ Log rotation

**Example strong password:**
```bash
# Generate secure password
openssl rand -base64 32
```

**Update `.env`:**
```bash
POSTGRES_PASSWORD=YOUR_SECURE_PASSWORD_HERE
DATABASE_URL=postgres://briefly:YOUR_SECURE_PASSWORD_HERE@localhost:5432/briefly?sslmode=require
```

---

## Summary

**Quick commands:**
```bash
make setup              # First time setup
make docker-up          # Start database
make migrate            # Run migrations
make feed-add URL=...  # Add feed
make feed-stats         # View statistics
make aggregate          # Fetch news
make docker-down        # Stop database
```

**Files:**
- `docker-compose.yml` - Development setup
- `docker-compose.prod.yml` - Production setup
- `.env` - Configuration (create from `.env.example`)
- `Makefile` - Convenient commands
- `docker/postgres/*.conf` - PostgreSQL tuning

**Services:**
- PostgreSQL: `localhost:5432`
- pgAdmin: `http://localhost:5050` (dev only)
- Redis: `localhost:6379` (full profile)

For more help, see:
- [NEWS_AGGREGATOR.md](./NEWS_AGGREGATOR.md) - Feature overview
- [MIGRATIONS.md](./MIGRATIONS.md) - Database migrations
- [PostgreSQL Docs](https://www.postgresql.org/docs/16/)
