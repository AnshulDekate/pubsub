# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for downloading dependencies)
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o chatroom .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 chatroom && \
    adduser -D -s /bin/sh -u 1000 -G chatroom chatroom

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/chatroom .

# Change ownership to non-root user
RUN chown chatroom:chatroom /app/chatroom

# Switch to non-root user
USER chatroom

# Expose port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

# Set environment variables
ENV PORT=9090
ENV GIN_MODE=release

# Run the application
CMD ["./chatroom"]
