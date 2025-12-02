# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod ./
COPY pkg/ ./pkg/
COPY service-2-article/go.mod service-2-article/go.sum ./service-2-article/

# Download dependencies
WORKDIR /build/service-2-article
RUN go mod download

# Copy source code
COPY service-2-article/ ./

# Tidy dependencies
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o article-service ./cmd/server/

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/service-2-article/article-service .

# Copy migrations if needed
COPY --from=builder /build/service-2-article/migrations ./migrations

# Expose gRPC port
EXPOSE 50052

# Run the application
CMD ["./article-service"]
