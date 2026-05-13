# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

# Runtime stage
FROM alpine:3.21

WORKDIR /app

# Install CA certificates for HTTPS connections (Supabase)
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/server .

# Copy migrations for potential runtime use
COPY --from=builder /app/migrations ./migrations

# Expose port (Render uses PORT env var)
EXPOSE 8080

# Run the binary
CMD ["./server"]
