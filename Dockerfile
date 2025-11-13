# Builder Base Stage
FROM golang:1.23.9-alpine AS builder-base
LABEL org.opencontainers.image.authors="reza@hashemi.dev"
ARG COMMIT_HASH
ARG BUILDNUMBER
ARG AUTHOR
ARG BUILD_TIMESTAMP
RUN mkdir -p /etc/ssl/certs/ && update-ca-certificates && apk add --no-cache git

# Builder Stage
FROM builder-base AS builder
WORKDIR /go/src/app
COPY . .
RUN go mod download
RUN go mod tidy

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    minor=$((BUILDNUMBER / 100)) && \
    patch=$((BUILDNUMBER % 100)) && \
    VERSION="v1.${minor}.${patch}" && \
    go build -ldflags "\
      -X 'github.com/getevo/evo/v2/lib/build.Commit=${COMMIT_HASH}' \
      -X 'github.com/getevo/evo/v2/lib/build.Version=${VERSION}' \
      -X 'github.com/getevo/evo/v2/lib/build.User=${AUTHOR}' \
      -s -w" \
      -o homa ./main.go

# Pre Runtime stage
FROM debian:bookworm-slim AS pre-runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
    rm -rf /var/lib/apt/lists/*

# Runtime stage
FROM pre-runtime
WORKDIR /app

# Copy binary from builder
COPY --from=builder /go/src/app/homa /app/homa

# Copy static files for Swagger UI and other static assets
COPY --from=builder /go/src/app/static /app/static

# Copy configuration files
COPY --from=builder /go/src/app/config.yml /app/config.yml

# Create non-root user for security
RUN groupadd -r homa && useradd -r -g homa homa
RUN chown -R homa:homa /app
USER homa

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

# Default command
CMD ["./homa"]