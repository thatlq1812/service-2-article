# Service 2: Article Service

> Content management microservice with user integration and pagination

**Protocol:** gRPC  
**Port:** 50052  
**Database:** PostgreSQL

---

## Table of Contents

- [Overview](#overview)
- [Setup Options](#setup-options)
  - [Option 1: Docker](#option-1-docker-recommended)
  - [Option 2: Terminal (Local)](#option-2-terminal-local-development)
- [Environment Configuration](#environment-configuration)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

---

## Overview

Article Service manages content creation and retrieval, integrating with User Service for author information.

**Features:**
- Article CRUD operations
- Author information integration (calls User Service)
- Pagination support
- Filter articles by user_id
- JWT authentication required for write operations

**Technology Stack:**
- **Language:** Go 1.21+
- **Protocol:** gRPC
- **Database:** PostgreSQL 15
- **Integration:** User Service gRPC client

---

## Setup Options

### Option 1: Docker (Recommended)

**Prerequisites:**
- Docker 20.10+
- Docker Compose 1.29+

**Quick Start:**
```bash
# From project root
cd agrios

# Configure environment
cp service-2-article/.env.example service-2-article/.env

# Start all services (includes Article Service)
docker-compose up -d

# Wait for services
sleep 15

# Run migrations
bash scripts/init-services.sh

# Verify service
docker-compose logs -f article-service
```

**Article Service Docker Details:**
```yaml
# From docker-compose.yml
article-service:
  build: ./service-2-article
  ports:
    - "50052:50052"
  depends_on:
    - postgres
    - user-service
  environment:
    - DB_HOST=postgres
    - USER_SERVICE_HOST=user-service
```

**Rebuild after code changes:**
```bash
docker-compose up -d --build article-service
```

---

### Option 2: Terminal (Local Development)

**Prerequisites:**
- Go 1.21+
- PostgreSQL 15+
- User Service running (for author info integration)

#### Step 1: Install Dependencies

```bash
cd service-2-article

# Download Go dependencies
go mod download

# Verify dependencies
go mod verify
```

#### Step 2: Setup Database

```bash
# Create database
psql -U postgres -c "CREATE DATABASE agrios_articles;"

# Run migration
psql -U postgres -d agrios_articles -f migrations/001_create_articles_table.sql

# Verify
psql -U postgres -d agrios_articles -c "\dt"
```

#### Step 3: Start User Service First

```bash
# Article Service needs User Service for author info
# In another terminal:
cd ../service-1-user
go run cmd/server/main.go

# Or use Docker for User Service only:
docker-compose up -d user-service
```

#### Step 4: Configure Environment

```bash
# Copy example
cp .env.example .env

# Edit configuration
nano .env
```

**Required settings for local development:**
```env
# Database (local PostgreSQL)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=agrios_articles

# User Service Integration
USER_SERVICE_HOST=localhost
USER_SERVICE_PORT=50051

# Server
GRPC_PORT=50052
```

#### Step 5: Build and Run

```bash
# Build
go build -o bin/article-service ./cmd/server

# Run
./bin/article-service

# Or run directly
go run cmd/server/main.go
```

**Expected output:**
```
2025/12/05 10:00:00 Connected to PostgreSQL
2025/12/05 10:00:00 Connected to User Service at localhost:50051
2025/12/05 10:00:00 Article Service listening on :50052
```

#### Step 6: Verify Service

```bash
# Install grpcurl (if not installed)
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List available services
grpcurl -plaintext localhost:50052 list

# Expected output:
# article.ArticleService
# grpc.reflection.v1alpha.ServerReflection
```

---

## Environment Configuration

### Complete Environment Variables

```env
# Database Configuration
DB_HOST=localhost               # PostgreSQL host (use 'postgres' for Docker)
DB_PORT=5432                    # PostgreSQL port
DB_USER=postgres                # Database user
DB_PASSWORD=yourpassword        # Database password
DB_NAME=agrios_articles         # Database name

# User Service Integration
USER_SERVICE_HOST=localhost     # User Service host (use 'user-service' for Docker)
USER_SERVICE_PORT=50051         # User Service port

# Server Configuration
GRPC_PORT=50052                 # gRPC server port
LOG_LEVEL=info                  # Logging level (debug, info, warn, error)
```

### Integration Notes

**User Service Dependency:**
- Article Service calls User Service to fetch author information
- GetArticle automatically includes author details
- If User Service is down, article data is still returned but author info may be missing
- Consider implementing circuit breaker for production

---

## API Reference

### gRPC Service Definition

```protobuf
service ArticleService {
  rpc CreateArticle (CreateArticleRequest) returns (CreateArticleResponse);
  rpc GetArticle (GetArticleRequest) returns (GetArticleResponse);
  rpc UpdateArticle (UpdateArticleRequest) returns (UpdateArticleResponse);
  rpc DeleteArticle (DeleteArticleRequest) returns (DeleteArticleResponse);
  rpc ListArticles (ListArticlesRequest) returns (ListArticlesResponse);
}
```

### 1. CreateArticle

Create a new article.

**Request:**
```bash
# Note: In production, user_id comes from JWT token
# For testing, provide user_id directly
grpcurl -plaintext \
  -d '{
    "user_id": 1,
    "title": "Introduction to Microservices",
    "content": "Microservices architecture is a design pattern that structures an application as a collection of loosely coupled services..."
  }' \
  localhost:50052 article.ArticleService.CreateArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "id": 1,
      "title": "Introduction to Microservices",
      "content": "Microservices architecture is...",
      "author": {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com"
      },
      "createdAt": "2025-12-05T10:00:00Z",
      "updatedAt": "2025-12-05T10:00:00Z"
    }
  }
}
```

**Validation:**
- Title: Required, min 3 characters
- Content: Required, min 10 characters
- User ID: Required, must exist in User Service

---

### 2. GetArticle

Retrieve article by ID with author information.

**Request:**
```bash
grpcurl -plaintext \
  -d '{"id": 1}' \
  localhost:50052 article.ArticleService.GetArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "id": 1,
      "title": "Introduction to Microservices",
      "content": "Microservices architecture is...",
      "author": {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com"
      },
      "createdAt": "2025-12-05T10:00:00Z",
      "updatedAt": "2025-12-05T10:00:00Z"
    }
  }
}
```

**Note:** Author information is fetched from User Service automatically

---

### 3. UpdateArticle

Update existing article (author only).

**Request:**
```bash
grpcurl -plaintext \
  -d '{
    "id": 1,
    "user_id": 1,
    "title": "Advanced Microservices Patterns",
    "content": "Updated content with advanced patterns..."
  }' \
  localhost:50052 article.ArticleService.UpdateArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "Article updated successfully",
  "data": {
    "article": {
      "id": 1,
      "title": "Advanced Microservices Patterns",
      "content": "Updated content with advanced patterns...",
      "author": {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com"
      },
      "updatedAt": "2025-12-05T11:00:00Z"
    }
  }
}
```

**Authorization:** Only the article author can update

---

### 4. DeleteArticle

Delete article (author only).

**Request:**
```bash
grpcurl -plaintext \
  -d '{
    "id": 1,
    "user_id": 1
  }' \
  localhost:50052 article.ArticleService.DeleteArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "Article deleted successfully"
}
```

**Authorization:** Only the article author can delete

---

### 5. ListArticles

Get paginated list of articles with optional user filter.

**Request:**
```bash
# List all articles (paginated)
grpcurl -plaintext \
  -d '{
    "page": 1,
    "page_size": 10
  }' \
  localhost:50052 article.ArticleService.ListArticles

# Filter by user_id
grpcurl -plaintext \
  -d '{
    "page": 1,
    "page_size": 10,
    "user_id": 1
  }' \
  localhost:50052 article.ArticleService.ListArticles
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "articles": [
      {
        "id": 1,
        "title": "Introduction to Microservices",
        "content": "Microservices architecture is...",
        "author": {
          "id": 1,
          "name": "John Doe",
          "email": "john@example.com"
        },
        "createdAt": "2025-12-05T10:00:00Z",
        "updatedAt": "2025-12-05T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 10,
      "total": 50
    }
  }
}
```

**Query Parameters:**
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 10, max: 100)
- `user_id`: Filter by author (optional)

---

## Database Schema

### Articles Table

```sql
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_articles_user_id ON articles(user_id);
CREATE INDEX idx_articles_created_at ON articles(created_at DESC);
```

**Note:** `user_id` is a foreign reference to User Service's users table (not enforced at DB level for service independence)

---

## Testing

### Unit Tests

```bash
cd service-2-article

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/repository/...

# Verbose output
go test -v ./...
```

### Integration Tests

```bash
# Ensure both services are running
docker-compose up -d user-service article-service

# Run test script
bash ../scripts/test-article-service.sh
```

### Manual Testing Workflow

```bash
# Prerequisites: User must exist in User Service
# 1. Create user in User Service
grpcurl -plaintext \
  -d '{"name":"Test User","email":"test@example.com","password":"pass123"}' \
  localhost:50051 user.UserService.CreateUser

# 2. Create article
grpcurl -plaintext \
  -d '{
    "user_id": 1,
    "title": "My First Article",
    "content": "This is the content of my first article."
  }' \
  localhost:50052 article.ArticleService.CreateArticle

# 3. Get article (with author info)
grpcurl -plaintext \
  -d '{"id": 1}' \
  localhost:50052 article.ArticleService.GetArticle

# 4. Update article
grpcurl -plaintext \
  -d '{
    "id": 1,
    "user_id": 1,
    "title": "Updated Title",
    "content": "Updated content."
  }' \
  localhost:50052 article.ArticleService.UpdateArticle

# 5. List articles
grpcurl -plaintext \
  -d '{"page": 1, "page_size": 10}' \
  localhost:50052 article.ArticleService.ListArticles

# 6. List user's articles only
grpcurl -plaintext \
  -d '{"page": 1, "page_size": 10, "user_id": 1}' \
  localhost:50052 article.ArticleService.ListArticles

# 7. Delete article
grpcurl -plaintext \
  -d '{"id": 1, "user_id": 1}' \
  localhost:50052 article.ArticleService.DeleteArticle

# 8. Verify deletion
grpcurl -plaintext \
  -d '{"id": 1}' \
  localhost:50052 article.ArticleService.GetArticle
# Expected: code "002" (not found)
```

---

## Troubleshooting

### Service Won't Start

**Problem:** Service fails to start

**Check logs:**
```bash
# Docker
docker-compose logs -f article-service

# Local
# Check terminal output for errors
```

**Common causes:**
1. Database not ready
2. User Service not accessible
3. Port 50052 already in use
4. Missing environment variables

**Solutions:**
```bash
# Wait for database
sleep 10

# Check PostgreSQL
docker-compose ps postgres
psql -h localhost -U postgres -d agrios_articles -c "SELECT 1;"

# Check User Service
docker-compose ps user-service
grpcurl -plaintext localhost:50051 list

# Check port
netstat -ano | findstr :50052  # Windows
lsof -i :50052                  # Linux/Mac
```

---

### Database Connection Failed

**Problem:** `failed to connect to postgres`

**Solutions:**
```bash
# 1. Verify database exists
psql -U postgres -l | grep agrios

# 2. Create database if missing
psql -U postgres -c "CREATE DATABASE agrios_articles;"

# 3. Run migrations
psql -U postgres -d agrios_articles -f migrations/001_create_articles_table.sql

# 4. Check credentials in .env
cat .env | grep DB_

# 5. Test connection
psql -h localhost -U postgres -d agrios_articles
```

---

### User Service Connection Failed

**Problem:** `failed to connect to user service` or author info missing

**Solutions:**
```bash
# 1. Check User Service is running
docker-compose ps user-service
# OR
curl localhost:50051  # Should not refuse connection

# 2. Verify USER_SERVICE_HOST in .env
cat .env | grep USER_SERVICE

# 3. Test User Service directly
grpcurl -plaintext localhost:50051 list

# 4. Check Docker network (if using Docker)
docker network inspect agrios_default

# 5. Restart Article Service after User Service is ready
docker-compose restart article-service
```

---

### Author Information Missing

**Problem:** GetArticle returns article but no author info

**Possible causes:**
1. User Service is down
2. User deleted from User Service
3. Network timeout

**Solutions:**
```bash
# 1. Check if user exists in User Service
grpcurl -plaintext \
  -d '{"id": 1}' \
  localhost:50051 user.UserService.GetUser

# 2. Check User Service logs
docker-compose logs user-service

# 3. Verify USER_SERVICE_HOST configuration
cat service-2-article/.env | grep USER_SERVICE

# 4. Test connection
grpcurl -plaintext localhost:50051 list
```

**Graceful Degradation:**
- Article data is still returned even if author info fails
- Check service logs for User Service connection errors
- Consider implementing circuit breaker in production

---

### Article Not Found

**Problem:** GetArticle returns `code "002"`

**Solutions:**
```bash
# 1. List all articles
grpcurl -plaintext \
  -d '{"page": 1, "page_size": 100}' \
  localhost:50052 article.ArticleService.ListArticles

# 2. Check database directly
psql -U postgres -d agrios_articles -c "SELECT * FROM articles;"

# 3. Verify article ID is correct
```

---

### Permission Denied

**Problem:** Cannot update/delete article (code "005")

**Cause:** user_id in request doesn't match article's author

**Solutions:**
```bash
# 1. Get article to check author
grpcurl -plaintext \
  -d '{"id": 1}' \
  localhost:50052 article.ArticleService.GetArticle

# 2. Use correct user_id
grpcurl -plaintext \
  -d '{
    "id": 1,
    "user_id": <correct_user_id>,
    "title": "Updated"
  }' \
  localhost:50052 article.ArticleService.UpdateArticle
```

---

## Project Structure

```
service-2-article/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── client/
│   │   └── user_client.go       # User Service gRPC client
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── db/
│   │   └── postgres.go          # PostgreSQL connection
│   ├── repository/
│   │   ├── article_repository.go # Interface
│   │   └── article_postgres.go   # Implementation
│   └── server/
│       └── article_server.go    # gRPC server implementation
├── proto/
│   ├── article_service.proto    # gRPC service definition
│   ├── article_service.pb.go    # Generated code
│   └── article_service_grpc.pb.go # Generated gRPC code
├── migrations/
│   └── 001_create_articles_table.sql
├── .env.example                 # Environment template
├── Dockerfile                   # Docker configuration
├── go.mod                       # Go dependencies
└── README.md                    # This file
```

---

## Development Commands

```bash
# Install dependencies
go mod download

# Update dependencies
go mod tidy

# Build
go build -o bin/article-service ./cmd/server

# Run
./bin/article-service

# Run with hot reload (requires air)
go install github.com/cosmtrek/air@latest
air

# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run

# Generate proto files
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/article_service.proto

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# View coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Service Integration

### Calling User Service

Article Service integrates with User Service to fetch author information:

```go
// internal/client/user_client.go
func (c *UserClient) GetUser(ctx context.Context, userID int32) (*userpb.User, error) {
    resp, err := c.client.GetUser(ctx, &userpb.GetUserRequest{
        Id: userID,
    })
    if err != nil {
        return nil, err
    }
    return resp.Data.User, nil
}
```

**Usage in GetArticle:**
```go
// 1. Get article from database
article, err := s.repo.GetArticle(ctx, req.Id)

// 2. Fetch author info from User Service
author, err := s.userClient.GetUser(ctx, article.UserID)

// 3. Combine and return
return &articlepb.GetArticleResponse{
    Code: "000",
    Message: "success",
    Data: &articlepb.GetArticleData{
        Article: &articlepb.Article{
            Id:      article.ID,
            Title:   article.Title,
            Content: article.Content,
            Author:  author,  // From User Service
        },
    },
}
```

---

## Additional Resources

- **[Main Project README](../README.md)** - Complete platform documentation
- **[Deployment Guide](../DEPLOYMENT.md)** - Production deployment
- **[Graceful Degradation Testing](./GRACEFUL_DEGRADATION_TESTING.md)** - Service resilience testing
- **[User Service](../service-1-user/README.md)** - Authentication service
- **[API Gateway](../service-3-gateway/README.md)** - REST API interface

---

**Service Version:** 1.0.0  
**Last Updated:** December 5, 2025  
**Maintainer:** thatlq1812@gmail.com
