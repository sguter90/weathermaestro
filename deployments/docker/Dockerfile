FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go.work and go.mod files first for better caching
COPY go.work .
COPY cmd/cli/go.* cmd/cli/
COPY pkg/database/go.* pkg/database/
COPY pkg/models/go.* pkg/models/
COPY pkg/pusher/go.* pkg/pusher/
COPY pkg/puller/go.* pkg/puller/

# Download dependencies (cached if go.mod/go.sum unchanged)
WORKDIR /app/cmd/cli
RUN go mod download

# Copy source code (only invalidates cache if code changes)
WORKDIR /app
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build the application
WORKDIR /app/cmd/cli
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /weathermaestro

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget && \
    addgroup -g 1000 weathermaestro && \
    adduser -D -u 1000 -G weathermaestro weathermaestro

# Copy binary and set ownership
COPY --from=builder /weathermaestro /usr/local/bin/weathermaestro
RUN chown weathermaestro:weathermaestro /usr/local/bin/weathermaestro

# Switch to non-root user
USER weathermaestro

EXPOSE 8059

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8059/health || exit 1

CMD ["/usr/local/bin/weathermaestro", "serve"]
