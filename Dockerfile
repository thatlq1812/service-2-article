# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy root go.mod and pkg folder
COPY go.mod go.sum* ./
COPY pkg/ ./pkg/

# Copy service-1-user (needed for proto files)
COPY service-1-user/ ./service-1-user/

# Copy service-2-article go.mod files
COPY service-2-article/go.mod service-2-article/go.sum ./service-2-article/

# Download dependencies
WORKDIR /build/service-2-article
RUN go mod download

# Copy source code
COPY service-2-article/ ./

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
