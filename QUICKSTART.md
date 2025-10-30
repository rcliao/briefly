# Quick Start Guide

Get up and running with Briefly in 5 minutes! 🚀

## Prerequisites

- Go 1.24+ installed
- Docker and Docker Compose installed
- Git

## Installation

### Option 1: With Docker (Recommended)

```bash
# 1. Clone repository
git clone https://github.com/rcliao/briefly.git
cd briefly

# 2. Copy environment file
cp .env.example .env

# 3. Edit .env and add your Gemini API key
vim .env
# Set: GEMINI_API_KEY=your-key-here

# 4. Start PostgreSQL
make docker-up

# 5. Build briefly
make build

# 6. Run migrations
make migrate

# 7. You're ready!
./briefly --help
```

### Option 2: Without Docker

```bash
# 1. Clone repository
git clone https://github.com/rcliao/briefly.git
cd briefly

# 2. Install PostgreSQL
brew install postgresql@16  # macOS
# or
sudo apt install postgresql-16  # Ubuntu

# 3. Create database
createdb briefly

# 4. Set connection string
export DATABASE_URL="postgres://localhost/briefly?sslmode=disable"

# 5. Copy and edit .env
cp .env.example .env
vim .env  # Add GEMINI_API_KEY

# 6. Build briefly
make build

# 7. Run migrations
make migrate
```

---

## First Steps

### 1. Add Your First Feed

```bash
# Add Hacker News (LLM filtered)
./briefly feed add "https://hnrss.org/newest?q=LLM+OR+GPT+OR+Claude"

# Add arXiv AI papers
./briefly feed add "https://arxiv.org/rss/cs.AI"

# List feeds
./briefly feed list
```

**Example output:**
```
ID          Title                Active  Last Fetched     Error Count
━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━  ━━━━━━  ━━━━━━━━━━━━━━  ━━━━━━━━━━━
abc123de... Hacker News: LLM     ✓       Never            0
def456gh... arXiv cs.AI          ✓       Never            0

Total feeds: 2
```

### 2. Aggregate News

```bash
# Fetch articles from all active feeds (last 24 hours)
./briefly aggregate --since 24
```

**Example output:**
```
📊 Aggregation Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Duration:           15.2s
Feeds Fetched:      2
Feeds Skipped:      0 (not modified)
Feeds Failed:       0
New Articles:       23
Duplicate Articles: 0

✅ Successfully aggregated 23 new articles
```

### 3. Check Feed Stats

```bash
# View statistics for all feeds
./briefly feed stats
```

**Example output:**
```
📊 Feed Statistics Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total Feeds:      2
Active Feeds:     2
Inactive Feeds:   0
Feeds with Errors: 0
Total Items:      23
```

### 4. Generate Digest (Manual)

```bash
# Create a markdown file with URLs
cat > input/weekly-links.md << EOF
# Weekly LLM News

- https://openai.com/blog/chatgpt-4
- https://www.anthropic.com/news/claude-2
- https://ai.google.dev/gemini-api
EOF

# Generate digest
./briefly digest input/weekly-links.md
```

---

## Useful Commands

### Docker

```bash
# Start database
make docker-up

# Start with pgAdmin
make docker-up-dev

# Stop database
make docker-down

# View logs
make docker-logs

# Open database shell
make db-shell
```

### Database

```bash
# Run migrations
make migrate

# Check migration status
make migrate-status

# Backup database
make backup

# Reset database (⚠️ deletes all data!)
make db-reset
```

### Feeds

```bash
# Add feed
make feed-add URL=https://example.com/feed.xml

# List feeds
make feed-list

# Aggregate news
make aggregate

# Disable a feed
./briefly feed disable <feed-id>

# Enable a feed
./briefly feed enable <feed-id>

# Remove a feed
./briefly feed remove <feed-id>
```

### Application

```bash
# Build
make build

# Run tests
make test

# Clean build artifacts
make clean

# View help
make help
```

---

## Daily Workflow

### As a Cron Job

```bash
# Edit crontab
crontab -e

# Add daily aggregation at 6am
0 6 * * * cd /path/to/briefly && ./briefly aggregate --since 24 >> /var/log/briefly.log 2>&1
```

### Manual Daily Run

```bash
# Each morning:
./briefly aggregate --since 24
./briefly feed stats

# Generate weekly digest (future feature):
./briefly digest --from-feeds
```

---

## Configuration

### Environment Variables

Edit `.env`:

```bash
# Required
GEMINI_API_KEY=your-key-here

# Database (Docker default)
DATABASE_URL=postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable

# Optional
LOG_LEVEL=info
DEBUG=false
```

### Configuration File

Edit `.briefly.yaml`:

```yaml
ai:
  gemini:
    api_key: ""  # Or use GEMINI_API_KEY env var
    model: "gemini-2.5-flash-preview-05-20"

database:
  connection_string: ""  # Or use DATABASE_URL env var
  max_connections: 25

cache:
  enabled: true
  directory: ".briefly-cache"
```

---

## Recommended Feeds

### LLM & AI News

```bash
# Hacker News (LLM filtered)
./briefly feed add "https://hnrss.org/newest?q=LLM+OR+GPT+OR+Claude+OR+AI"

# arXiv Computer Science - AI
./briefly feed add "https://arxiv.org/rss/cs.AI"

# OpenAI Blog
./briefly feed add "https://openai.com/blog/rss/"

# Google AI Blog
./briefly feed add "https://blog.google/technology/ai/rss/"

# Hugging Face Blog
./briefly feed add "https://huggingface.co/blog/feed.xml"

# MIT Technology Review - AI
./briefly feed add "https://www.technologyreview.com/topic/artificial-intelligence/feed"
```

### Tech News

```bash
# TechCrunch
./briefly feed add "https://techcrunch.com/feed/"

# The Verge
./briefly feed add "https://www.theverge.com/rss/index.xml"

# Ars Technica
./briefly feed add "https://feeds.arstechnica.com/arstechnica/index"
```

---

## Troubleshooting

### Database Connection Failed

```bash
# Check if PostgreSQL is running
docker-compose ps

# Check logs
docker-compose logs postgres

# Restart database
make docker-down
make docker-up
```

### Migration Failed

```bash
# Check migration status
make migrate-status

# View migration logs
./briefly migrate up 2>&1 | tee migration.log

# Reset database (⚠️ deletes data)
make db-reset
```

### Feed Fetch Failed

```bash
# Check feed details
./briefly feed stats <feed-id>

# Test feed URL manually
curl -I https://example.com/feed.xml

# Remove and re-add feed
./briefly feed remove <feed-id>
./briefly feed add https://example.com/feed.xml
```

---

## Next Steps

1. **Set up daily cron job** - Automate news aggregation
2. **Add more feeds** - Curate your LLM news sources
3. **Explore features** - Check `docs/NEWS_AGGREGATOR.md`
4. **Deploy to cloud** - See `docs/DOCKER.md` for production setup

---

## Resources

- **Full Documentation**: `docs/NEWS_AGGREGATOR.md`
- **Database Migrations**: `docs/MIGRATIONS.md`
- **Docker Setup**: `docs/DOCKER.md`
- **Commands**: `make help`

---

## Getting Help

```bash
# View all commands
./briefly --help

# Command-specific help
./briefly feed --help
./briefly aggregate --help
./briefly migrate --help

# Makefile commands
make help
```

---

## Example Session

```bash
# Complete first-time setup
$ make setup
Starting PostgreSQL...
✅ PostgreSQL started on localhost:5432
Running migrations...
✅ All migrations applied successfully
✅ Setup complete!

# Add feeds
$ make feed-add URL=https://hnrss.org/newest?q=LLM
✅ Feed added successfully
   ID:    abc123de-4567-89ab-cdef-0123456789ab
   Title: Hacker News: LLM

# Aggregate news
$ make aggregate
📊 Aggregation Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Duration:         12.5s
Feeds Fetched:    1
New Articles:     15
✅ Successfully aggregated 15 new articles

# Check stats
$ make feed-list
ID          Title              Active  Last Fetched         Error Count
━━━━━━━━━━  ━━━━━━━━━━━━━━━━  ━━━━━━  ━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━
abc123de... Hacker News: LLM   ✓       2025-10-24 15:30     0

Total feeds: 1

# You're all set! 🎉
```

---

## What's Next?

Briefly is actively being developed. Upcoming features:

- 🔜 **Web search integration** - Hybrid RSS + search
- 🔜 **Web interface** - Browse news in browser
- 🔜 **Digest from feeds** - `briefly digest --from-feeds`
- 🔜 **API mode** - RESTful API for frontend

Stay tuned! ⭐ Star the repo for updates.
