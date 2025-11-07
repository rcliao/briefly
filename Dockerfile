# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o briefly \
    ./cmd/briefly

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S briefly && \
    adduser -u 1001 -S briefly -G briefly

# Set working directory
WORKDIR /home/briefly

# Copy binary from builder
COPY --from=builder /app/briefly .

# Copy web assets (templates and static files)
COPY --from=builder /app/web ./web

# Change ownership to non-root user
RUN chown -R briefly:briefly /home/briefly

# Switch to non-root user
USER briefly

# Expose port (Railway will override with PORT env var)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health || exit 1

# Run the server
CMD ["./briefly", "serve"]
