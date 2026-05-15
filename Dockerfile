# syntax=docker/dockerfile:1

# -----------------------------
# Build stage
# -----------------------------
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build binary (static, stripped)
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/victorhsb/branchless-pr/internal/cli.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -trimpath \
    -o stack-pr \
    ./cmd/stack-pr

# -----------------------------
# Runtime stage
# -----------------------------
FROM alpine:3.21
RUN apk add --no-cache \
    git \
    github-cli \
    ca-certificates \
    openssh-client

COPY --from=builder /app/stack-pr /usr/local/bin/stack-pr

LABEL org.opencontainers.image.source="https://github.com/victorhsb/branchless-pr"
LABEL org.opencontainers.image.description="CLI for stacked GitHub PRs"
LABEL org.opencontainers.image.licenses="Apache-2.0"

ENTRYPOINT ["stack-pr"]
