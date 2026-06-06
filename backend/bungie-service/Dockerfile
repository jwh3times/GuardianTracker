# Build stage with CGO for SQLite support
FROM golang:1.25-alpine AS builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application WITH CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o main .

# Runtime stage - use specific minimal alpine image
FROM alpine:3.19

# Install SQLite runtime library, ca-certificates, and create non-root user
RUN apk --no-cache add ca-certificates sqlite-libs && \
    addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Create data directory for manifest with proper ownership
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port 8082
EXPOSE 8082

# Volume for manifest persistence
VOLUME ["/app/data"]

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

# Command to run the executable
CMD ["./main"]
