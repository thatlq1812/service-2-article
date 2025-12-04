# Article Service

gRPC microservice for article management with user integration.

**Port:** `50052` | **Protocol:** gRPC

## Quick Start

```bash
# Setup database
psql -U postgres -c "CREATE DATABASE agrios_articles;"
psql -U postgres -d agrios_articles -f migrations/001_create_articles_table.sql

# Run (User Service must be running first!)
cp .env.example .env  # Edit with your config
go run cmd/server/main.go
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_NAME` | agrios_articles | Database name |
| `REDIS_ADDR` | localhost:6379 | Redis (shared with User Service) |
| `JWT_SECRET` | *required* | JWT secret (must match User Service) |
| `USER_SERVICE_ADDR` | localhost:50051 | User Service address |
| `GRPC_PORT` | 50052 | gRPC server port |

---

# API Reference

### 1. CreateArticle

Create a new article. **Requires authentication.**

**Request:**
```protobuf
message CreateArticleRequest {
  string title = 1;    // Required
  string content = 2;  // Required
  int32 user_id = 3;   // Required (author)
}
```

**Example:**
```bash
grpcurl -plaintext \
  -H "authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{
    "title": "My First Article",
    "content": "This is the content...",
    "user_id": 1
  }' localhost:50052 article.ArticleService/CreateArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "id": 1,
      "title": "My First Article",
      "content": "This is the content...",
      "user_id": 1,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  }
}
```

---

### 2. GetArticle

Get article with author information.

**Request:**
```protobuf
message GetArticleRequest {
  int32 id = 1;  // Required
}
```

**Example:**
```bash
grpcurl -plaintext -d '{"id": 1}' localhost:50052 article.ArticleService/GetArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "article": {
        "id": 1,
        "title": "My First Article",
        "content": "This is the content...",
        "user_id": 1,
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      },
      "user": {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com"
      }
    }
  }
}
```

---

### 3. UpdateArticle

Update article title and/or content.

**Request:**
```protobuf
message UpdateArticleRequest {
  int32 id = 1;       // Required
  string title = 2;   // Optional
  string content = 3; // Optional
}
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": 1,
  "title": "Updated Title",
  "content": "Updated content..."
}' localhost:50052 article.ArticleService/UpdateArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "id": 1,
      "title": "Updated Title",
      "content": "Updated content...",
      "user_id": 1,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T12:00:00Z"
    }
  }
}
```

---

### 4. DeleteArticle

Delete an article.

**Request:**
```protobuf
message DeleteArticleRequest {
  int32 id = 1;  // Required
}
```

**Example:**
```bash
grpcurl -plaintext -d '{"id": 1}' localhost:50052 article.ArticleService/DeleteArticle
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "success": true
  }
}
```

---

### 5. ListArticles

Get paginated article list with optional user filter.

**Request:**
```protobuf
message ListArticlesRequest {
  int32 page_size = 1;    // Default: 10
  int32 page_number = 2;  // Default: 1
  int32 user_id = 3;      // Optional: filter by author (0 = all)
}
```

**Example:**
```bash
# List all articles
grpcurl -plaintext -d '{
  "page_size": 10,
  "page_number": 1
}' localhost:50052 article.ArticleService/ListArticles

# Filter by author
grpcurl -plaintext -d '{
  "page_size": 10,
  "page_number": 1,
  "user_id": 1
}' localhost:50052 article.ArticleService/ListArticles
```

**Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "articles": [
      {
        "article": {
          "id": 1,
          "title": "First Article",
          "content": "Content...",
          "user_id": 1
        },
        "user": {
          "id": 1,
          "name": "John Doe",
          "email": "john@example.com"
        }
      }
    ],
    "total": 25,
    "page": 1,
    "total_pages": 3
  }
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| `000` | Success |
| `001` | Validation error |
| `002` | User not found (author) |
| `003` | Article not found |
| `005` | Unauthorized (invalid/missing token) |
| `500` | Internal server error |

---

## Service Dependencies

- **User Service** - JWT validation, user data enrichment
- **Redis** - Token blacklist (shared with User Service)

If User Service unavailable: GetArticle returns article without user data.
