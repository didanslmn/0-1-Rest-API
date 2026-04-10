# Build stage
FROM golang:alpine AS builder

# Mempercepat download dependensi dengan proxy alternatif
ENV GOPROXY=https://goproxy.io,direct

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

# Run stage
FROM alpine:3.19

WORKDIR /app

# Install CA certificates for secure database connections (crucial for Neon DB)
RUN apk --no-cache add ca-certificates

# Buat user non-root untuk keamanan
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary dari stage builder
COPY --from=builder /app/main .

# Ganti ownership ke user non-root
RUN chown -R appuser:appgroup /app

USER appuser

# Expose port
EXPOSE 8080

# Healthcheck bawaan Docker
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Run the binary
CMD ["./main"]
