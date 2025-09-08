# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install necessary packages
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o domain-exporter .

# Runtime stage
FROM alpine:latest

# Install ca certificates and set timezone to Hong Kong
RUN apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Hong_Kong /etc/localtime && \
    echo "Asia/Hong_Kong" > /etc/timezone

WORKDIR /root/

# Copy binary from build stage
COPY --from=builder /app/domain-exporter .
COPY --from=builder /app/config.yaml .

# Expose port
EXPOSE 8080

# Run application
CMD ["./domain-exporter", "-config=config.yaml"]