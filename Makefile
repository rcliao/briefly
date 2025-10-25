# Briefly Makefile
# Convenient commands for development and deployment

.PHONY: help build test clean docker-up docker-down docker-logs migrate db-shell

# Default target
help:
	@echo "Briefly - Available Commands:"
	@echo ""
	@echo "  Development:"
	@echo "    make build              Build the briefly binary"
	@echo "    make test               Run all tests"
	@echo "    make clean              Clean build artifacts"
	@echo "    make run                Run briefly locally"
	@echo ""
	@echo "  Docker:"
	@echo "    make docker-up          Start PostgreSQL with Docker"
	@echo "    make docker-up-dev      Start PostgreSQL + pgAdmin"
	@echo "    make docker-up-full     Start PostgreSQL + pgAdmin + Redis"
	@echo "    make docker-down        Stop all Docker containers"
	@echo "    make docker-logs        Show Docker logs"
	@echo "    make docker-clean       Remove Docker volumes (⚠️  deletes data)"
	@echo ""
	@echo "  Database:"
	@echo "    make migrate            Run database migrations"
	@echo "    make migrate-status     Check migration status"
	@echo "    make db-shell           Open PostgreSQL shell"
	@echo "    make db-reset           Reset database (⚠️  deletes all data)"
	@echo ""
	@echo "  Feeds:"
	@echo "    make feed-add URL=...  Add an RSS feed"
	@echo "    make feed-list          List all feeds"
	@echo "    make feed-stats         Show feed statistics"
	@echo "    make aggregate          Run news aggregation"
	@echo ""
	@echo "  Production:"
	@echo "    make docker-up-prod     Start production PostgreSQL"
	@echo "    make backup             Backup database"

# Build
build:
	@echo "Building briefly..."
	go build -o briefly ./cmd/briefly
	@echo "✅ Build complete: ./briefly"

# Test
test:
	@echo "Running tests..."
	go test ./... -v

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f briefly
	go clean
	@echo "✅ Clean complete"

# Run
run: build
	./briefly

# Docker Development
docker-up:
	@echo "Starting PostgreSQL..."
	docker-compose up -d postgres
	@echo "✅ PostgreSQL started on localhost:5432"
	@echo ""
	@echo "Connection string:"
	@echo "  DATABASE_URL=postgres://briefly:briefly_dev_password@localhost:5432/briefly?sslmode=disable"

docker-up-dev:
	@echo "Starting PostgreSQL + pgAdmin..."
	docker-compose --profile dev up -d
	@echo "✅ Services started:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  pgAdmin:    http://localhost:5050"
	@echo ""
	@echo "pgAdmin credentials:"
	@echo "  Email:    admin@briefly.local"
	@echo "  Password: admin"

docker-up-full:
	@echo "Starting all services..."
	docker-compose --profile full up -d
	@echo "✅ All services started"

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down
	@echo "✅ Containers stopped"

docker-logs:
	docker-compose logs -f

docker-clean:
	@echo "⚠️  WARNING: This will delete all database data!"
	@read -p "Are you sure? (yes/no): " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		docker-compose down -v; \
		echo "✅ Volumes removed"; \
	else \
		echo "Cancelled"; \
	fi

# Docker Production
docker-up-prod:
	@echo "Starting production PostgreSQL..."
	@if [ ! -f .env ]; then \
		echo "❌ Error: .env file not found"; \
		echo "Copy .env.example to .env and set POSTGRES_PASSWORD"; \
		exit 1; \
	fi
	docker-compose -f docker-compose.prod.yml up -d
	@echo "✅ Production PostgreSQL started"

# Database
migrate: build
	@echo "Running database migrations..."
	./briefly migrate up

migrate-status: build
	@echo "Checking migration status..."
	./briefly migrate status

db-shell:
	@echo "Opening PostgreSQL shell..."
	docker-compose exec postgres psql -U briefly -d briefly

db-reset: docker-clean docker-up migrate
	@echo "✅ Database reset complete"

# Feeds
feed-add: build
	@if [ -z "$(URL)" ]; then \
		echo "❌ Error: URL not specified"; \
		echo "Usage: make feed-add URL=https://example.com/feed.xml"; \
		exit 1; \
	fi
	./briefly feed add $(URL)

feed-list: build
	./briefly feed list

feed-stats: build
	./briefly feed stats

aggregate: build
	./briefly aggregate --since 24

# Backup
backup:
	@echo "Creating database backup..."
	@mkdir -p backups
	docker-compose exec -T postgres pg_dump -U briefly -d briefly > backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup created in backups/"

# Setup (first time)
setup: docker-up
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@echo "Running migrations..."
	@$(MAKE) migrate
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Add your first feed:  make feed-add URL=https://hnrss.org/newest"
	@echo "  2. Aggregate news:       make aggregate"
	@echo "  3. Check feeds:          make feed-list"
	@echo "  4. View statistics:      make feed-stats"
