# Multi-stage Dockerfile for LinkGen AI

# Development stage with hot reload
FROM golang:1.21-alpine AS development

WORKDIR /app

# Install development tools
RUN go install github.com/cosmtrek/air@latest

# Copy go mod files
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./

# Expose application port
EXPOSE 8080

# Run with air for hot reload
CMD ["air", "-c", ".air.toml"]

# Builder stage for production
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o linkgenai main.go

# Production stage
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/linkgenai .

# Expose application port
EXPOSE 8080

# Run the application
CMD ["./linkgenai"]
