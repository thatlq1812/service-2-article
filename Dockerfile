# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go.mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o article-service ./cmd/server/

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/article-service .

# Copy migrations if needed
COPY --from=builder /build/migrations ./migrations

# Expose gRPC port
EXPOSE 50052

# Run the application
CMD ["./article-service"]
