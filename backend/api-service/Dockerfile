# Build stage — CGO required for SQLite
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o main .

# Runtime stage
FROM alpine:3.24

RUN apk --no-cache add ca-certificates sqlite-libs && \
    addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

WORKDIR /app

RUN mkdir -p /app/data && chown -R appuser:appgroup /app

COPY --from=builder /app/main .

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8081

VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

CMD ["./main"]
