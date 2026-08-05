# Briefly Makefile

.PHONY: help build test lint fmt vet clean run

help:
	@echo "Briefly - Available Commands:"
	@echo ""
	@echo "    make build              Build the briefly binary"
	@echo "    make test               Run all tests (with -race)"
	@echo "    make lint               Run golangci-lint"
	@echo "    make fmt                Format all Go files"
	@echo "    make vet                Run go vet"
	@echo "    make clean              Clean build artifacts"
	@echo "    make run                Build and run briefly"

build:
	@echo "Building briefly..."
	go build -o briefly ./cmd/briefly
	@echo "✅ Build complete: ./briefly"

test:
	go test -race ./...

# Lint (matches CI; install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)
lint:
	$(shell go env GOPATH)/bin/golangci-lint run

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -f briefly
	go clean
	@echo "✅ Clean complete"

run: build
	./briefly
