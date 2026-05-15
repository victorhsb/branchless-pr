# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o stack-pr ./cmd/stack-pr

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache git github-cli ca-certificates
COPY --from=builder /app/stack-pr /usr/local/bin/stack-pr
ENTRYPOINT ["stack-pr"]
