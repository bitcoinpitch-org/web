# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with mod=readonly to ignore vendor directory
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -o /app/bin/server ./cmd/server
# Build the migration binary
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -o /app/bin/migrate ./cmd/migrate
# Build the admin token generator
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -o /app/bin/admin-token ./cmd/admin-token

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/server /app/server
# Copy migration binary from builder
COPY --from=builder /app/bin/migrate /app/migrate
# Copy admin token generator from builder
COPY --from=builder /app/bin/admin-token /app/admin-token

# Copy static files
COPY --from=builder /app/static /app/static

# Copy templates - Fixed: preserve internal/templates structure
COPY --from=builder /app/internal/templates /app/internal/templates

# Copy i18n translation files
COPY --from=builder /app/i18n /app/i18n

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8090

# Run the application
CMD ["/app/server"] 