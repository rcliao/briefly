#!/bin/bash
# Setup Local Development Database with pgvector
# This script automates the local setup with zero manual work

set -e

echo "🚀 Setting up Local Development Database with pgvector"
echo "======================================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Check if Docker is running
echo "1️⃣  Checking Docker..."
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    echo "   Please start Docker Desktop and try again"
    exit 1
fi
echo -e "${GREEN}   ✅ Docker is running${NC}"
echo ""

# Step 2: Check if existing container is running
echo "2️⃣  Checking for existing database..."
if docker ps | grep -q "briefly-postgres"; then
    echo -e "${YELLOW}   ⚠️  Existing database is running${NC}"
    read -p "   Stop and remove existing database? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker-compose down
        echo -e "${GREEN}   ✅ Stopped existing database${NC}"
    else
        echo "   Cancelled by user"
        exit 0
    fi
else
    echo "   ℹ️  No existing database found"
fi
echo ""

# Step 3: Check if data volume exists
echo "3️⃣  Checking for existing data volume..."
if docker volume ls | grep -q "briefly_postgres_data"; then
    echo -e "${YELLOW}   ⚠️  Existing data volume found${NC}"
    echo "   This volume contains your database data"
    read -p "   Remove existing data and start fresh? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker volume rm briefly_postgres_data
        echo -e "${GREEN}   ✅ Removed existing data volume${NC}"
    else
        echo "   ℹ️  Keeping existing data volume"
    fi
else
    echo "   ℹ️  No existing data volume found"
fi
echo ""

# Step 4: Start new database with pgvector
echo "4️⃣  Starting database with pgvector..."
docker-compose up -d postgres

# Wait for database to be ready
echo "   ⏳ Waiting for database to be ready..."
for i in {1..30}; do
    if docker exec briefly-postgres pg_isready -U briefly -d briefly > /dev/null 2>&1; then
        echo -e "${GREEN}   ✅ Database is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}   ❌ Database failed to start${NC}"
        echo "   Check logs: docker-compose logs postgres"
        exit 1
    fi
    sleep 1
done
echo ""

# Step 5: Verify pgvector is available
echo "5️⃣  Verifying pgvector availability..."
PGVECTOR_AVAILABLE=$(docker exec briefly-postgres psql -U briefly -d briefly -tAc "SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name='vector');")

if [ "$PGVECTOR_AVAILABLE" = "t" ]; then
    echo -e "${GREEN}   ✅ pgvector extension is available${NC}"
else
    echo -e "${RED}   ❌ pgvector extension not available${NC}"
    echo "   This shouldn't happen with pgvector/pgvector image"
    exit 1
fi
echo ""

# Step 6: Run migrations
echo "6️⃣  Running database migrations..."
if [ ! -f "./briefly" ]; then
    echo -e "${YELLOW}   ⚠️  briefly binary not found${NC}"
    echo "   Building briefly..."
    make build
fi

export DATABASE_URL="postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"
./briefly migrate up

echo ""

# Step 7: Verify pgvector is enabled
echo "7️⃣  Verifying pgvector extension is enabled..."
PGVECTOR_ENABLED=$(docker exec briefly-postgres psql -U briefly -d briefly -tAc "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='vector');")

if [ "$PGVECTOR_ENABLED" = "t" ]; then
    PGVECTOR_VERSION=$(docker exec briefly-postgres psql -U briefly -d briefly -tAc "SELECT extversion FROM pg_extension WHERE extname='vector';")
    echo -e "${GREEN}   ✅ pgvector v${PGVECTOR_VERSION} is enabled${NC}"
else
    echo -e "${YELLOW}   ⚠️  pgvector not enabled (migration 002 may have been skipped)${NC}"
fi
echo ""

# Step 8: Check if vector column exists
echo "8️⃣  Checking vector column..."
VECTOR_COLUMN=$(docker exec briefly-postgres psql -U briefly -d briefly -tAc "SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='articles' AND column_name='embedding_vector');")

if [ "$VECTOR_COLUMN" = "t" ]; then
    echo -e "${GREEN}   ✅ embedding_vector column exists${NC}"
else
    echo "   ℹ️  embedding_vector column not created yet (will be created when you add articles)"
fi
echo ""

# Summary
echo "════════════════════════════════════════════════════════"
echo -e "${GREEN}✅ Local Development Setup Complete!${NC}"
echo "════════════════════════════════════════════════════════"
echo ""
echo "📊 Database Info:"
echo "   • Image: pgvector/pgvector:pg16-alpine"
echo "   • Host: localhost:5432"
echo "   • Database: briefly"
echo "   • User: briefly"
echo "   • Password: briefly_dev_password"
echo ""
echo "🔌 Connection:"
echo "   DATABASE_URL=postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"
echo ""
echo "🛠️  Management:"
echo "   • View logs: docker-compose logs postgres"
echo "   • Stop database: docker-compose down"
echo "   • Start database: docker-compose up -d"
echo "   • Access psql: docker exec -it briefly-postgres psql -U briefly -d briefly"
echo ""
echo "🚀 Next Steps:"
echo "   1. Add some RSS feeds: ./briefly feed add <url>"
echo "   2. Aggregate articles: ./briefly aggregate --since 24"
echo "   3. Generate digest: ./briefly digest generate --since 7"
echo ""
echo "📖 For more info, see: PGVECTOR_SETUP.md"
