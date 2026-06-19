# Build stage
FROM golang:1.26.4-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary. APP_VERSION is the top released CHANGELOG version (SemVer source
# of truth), injected into internal/version; defaults to "dev" for plain builds.
ARG APP_VERSION=dev
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
	-ldflags "-X github.com/Sternrassler/eve-o-provit/backend/internal/version.Version=${APP_VERSION}" \
	-o api ./cmd/api

# Runtime stage — pinned (not :latest) for reproducible builds.
FROM alpine:3.21

RUN apk --no-cache add ca-certificates sqlite-libs wget

# Run as non-root (hardening). The api writes nothing to disk — it reads the
# read-only SDE mount and talks to postgres/redis; /tmp is a tmpfs at runtime —
# so a plain unprivileged user needs no writable paths in the image.
RUN adduser -D -u 10001 appuser
WORKDIR /app

# Copy binary from builder, owned by the non-root user
COPY --from=builder --chown=appuser:appuser /app/api .

USER appuser

EXPOSE 8080

CMD ["./api"]
