# Article Service

Article management microservice for Agrios platform with user integration.

## Features

- Article CRUD operations
- User authentication via JWT
- Automatic user data enrichment (joins with User Service)
- gRPC API with wrapped response format
- Microservice-to-microservice communication

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- **User Service running** (for user data integration)

### Setup

1. **Clone and configure:**
```bash
git clone <repo-url>
cd service-2-article
cp .env.example .env
# Edit .env with your configuration
```

2. **Install dependencies:**
```bash
go mod download
```

3. **Setup database:**
```bash
# Create database
psql -U postgres -c "CREATE DATABASE agrios_articles;"

# Run migrations
psql -U postgres -d agrios_articles -f migrations/001_create_articles_table.sql
```

4. **Run service:**
```bash
# Make sure User Service is running first!
# Development
go run cmd/server/main.go

# Build and run
go build -o bin/article-service cmd/server/main.go
./bin/article-service
```

Service will start on port **50052** (gRPC).

### Docker

```bash
# Build
docker build -t article-service .

# Run (link to user service)
docker run -p 50052:50052 --env-file .env article-service
```

## API Documentation

### gRPC Methods

1. **CreateArticle** - Create new article (requires auth)
2. **GetArticle** - Get article by ID (with user info)
3. **UpdateArticle** - Update article
4. **DeleteArticle** - Delete article
5. **ListArticles** - Paginated article list (with user info)

### Authentication

CreateArticle requires JWT token in metadata:

```bash
grpcurl -plaintext \
  -H "authorization: Bearer <token>" \
  -d '{"title":"My Article","content":"Content..."}' \
  localhost:50052 article.ArticleService.CreateArticle
```

### Response Format

All responses use wrapped format:
```json
{
  "code": "000",
  "message": "success",
  "data": {...}
}
```

**Error codes:**
- `000` - Success
- `003` - Invalid argument (missing title/content)
- `005` - Not found
- `013` - Internal error
- `014` - Unauthenticated (missing/invalid token)

### User Data Integration

GetArticle and ListArticles automatically fetch user information from User Service:

```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "article": {
        "id": 1,
        "title": "My Article",
        "userId": 1
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

### Testing

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 1. Login via User Service to get token
TOKEN=$(grpcurl -plaintext \
  -d '{"email":"user@example.com","password":"pass123"}' \
  localhost:50051 user.UserService.Login \
  | grep 'accessToken' | cut -d'"' -f4)

# 2. Create article with auth
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"title":"Test Article","content":"Content here"}' \
  localhost:50052 article.ArticleService.CreateArticle

# 3. Get article (no auth needed)
grpcurl -plaintext -d '{"id":1}' \
  localhost:50052 article.ArticleService.GetArticle
```

## Configuration

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | postgres | Database user |
| `DB_PASSWORD` | postgres | Database password |
| `DB_NAME` | agrios_articles | Database name |
| `USER_SERVICE_ADDR` | localhost:50051 | User Service gRPC address |
| `GRPC_PORT` | 50052 | gRPC server port |

## Project Structure

```
service-2-article/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── auth/                   # JWT validation
│   ├── client/                 # User Service gRPC client
│   ├── config/                 # Configuration loading
│   ├── db/                     # Database connection
│   ├── repository/             # Data access layer
│   ├── response/               # Response helpers
│   └── server/                 # gRPC server implementation
├── proto/                      # Protocol buffer definitions
├── migrations/                 # Database migrations
├── Dockerfile                  # Container image
└── .env.example               # Environment template
```

## Development

### Generate Proto Files

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/article_service.proto
```

### Database Migrations

Migrations are in `migrations/` directory. Apply manually:

```bash
psql -U postgres -d agrios_articles -f migrations/001_create_articles_table.sql
```

## Dependencies

**Service Dependencies:**
- User Service must be running for:
  - Article creation (JWT validation)
  - User data enrichment (GetArticle, ListArticles)

**Graceful Degradation:**
- If User Service is unavailable, GetArticle returns article with empty user data
- Article creation will fail (authentication required)

## Troubleshooting

**User Service connection error:**
```
ERROR: failed to connect to user service
```
- Check User Service is running: `grpcurl -plaintext localhost:50051 list`
- Verify `USER_SERVICE_ADDR` in .env

**Authentication required:**
```json
{"code":"014","message":"authentication required"}
```
- CreateArticle needs JWT token in metadata header
- Get token from User Service Login endpoint

**Database connection refused:**
- Check PostgreSQL is running: `pg_isready`
- Verify database exists: `psql -U postgres -l | grep agrios_articles`

**Port already in use:**
```bash
# Find process using port 50052
netstat -ano | findstr :50052
# Kill or change GRPC_PORT in .env
```

## License

MIT License
