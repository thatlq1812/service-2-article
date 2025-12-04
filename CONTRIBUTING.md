# Contributing to Article Service

Thank you for considering contributing to the Article Service!

## Development Setup

1. **Fork and clone:**
```bash
git clone https://github.com/YOUR_USERNAME/service-2-article.git
cd service-2-article
```

2. **Setup environment:**
```bash
./setup.sh
```

3. **Start User Service** (required dependency):
```bash
# In another terminal
cd ../service-1-user
./bin/user-service
```

4. **Create feature branch:**
```bash
git checkout -b feature/your-feature-name
```

## Code Standards

### Go Style
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run linter: `golangci-lint run`

### Commit Messages
Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add article tags support
fix: handle user service unavailable gracefully
docs: update API documentation
chore: upgrade dependencies
test: add tests for CreateArticle
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance tasks

### Response Format
All new endpoints must use wrapped response format:
```go
return &pb.CreateArticleResponse{
    Code:    response.MapGRPCCodeToString(codes.OK),
    Message: "success",
    Data:    &pb.CreateArticleResponseData{Article: article},
}, nil
```

## Testing

### Unit Tests
```bash
go test ./...
```

### Integration Tests
```bash
# Start dependencies
docker-compose up -d postgres

# Start User Service in another terminal
cd ../service-1-user && go run cmd/server/main.go

# Run tests
go test ./... -tags=integration
```

### Manual Testing
```bash
# 1. Start service
go run cmd/server/main.go

# 2. Get auth token from User Service
TOKEN=$(grpcurl -plaintext \
  -d '{"email":"test@example.com","password":"pass"}' \
  localhost:50051 user.UserService.Login \
  | grep 'accessToken' | cut -d'"' -f4)

# 3. Test article creation
grpcurl -plaintext -H "authorization: Bearer $TOKEN" \
  -d '{"title":"Test","content":"Content"}' \
  localhost:50052 article.ArticleService.CreateArticle
```

## Pull Request Process

1. **Update documentation** - If adding features, update README.md
2. **Add tests** - Include unit tests for new code
3. **Update CHANGELOG.md** - Document your changes
4. **Clean commits** - Squash WIP commits
5. **Pass CI** - Ensure all checks pass

### PR Template
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Dependencies
- [ ] Requires User Service changes
- [ ] Independent change

## Testing
How was this tested?

## Checklist
- [ ] Code follows project style
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Tested with User Service integration
```

## Database Migrations

**Adding new migration:**
1. Create file: `migrations/XXX_description.sql`
2. Test locally: `psql -U postgres -d agrios_articles -f migrations/XXX_description.sql`
3. Document in README.md

**Migration guidelines:**
- Always provide rollback steps
- Test on sample data
- Consider foreign key relationships

## Proto Changes

**Modifying .proto files:**
1. Edit `proto/article_service.proto`
2. Regenerate code:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/article_service.proto
```
3. Update affected handlers
4. Maintain backward compatibility if possible

## User Service Integration

**When adding features that need user data:**
1. Use existing `user_client.go` for gRPC calls
2. Handle User Service unavailability gracefully
3. Implement circuit breaker if needed
4. Add timeout configuration

**Example:**
```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

userResp, err := s.userClient.GetUser(ctx, &userpb.GetUserRequest{Id: userId})
if err != nil {
    // Graceful degradation - return article without user data
    log.Printf("failed to get user: %v", err)
}
```

## Authentication

**Adding authenticated endpoints:**
1. Extract token from metadata:
```go
token := auth.ExtractToken(ctx)
if token == "" {
    return response.CreateArticleError(codes.Unauthenticated, "authentication required"), nil
}
```
2. Validate via User Service
3. Use user ID from validated token

## Security

**Reporting vulnerabilities:**
- DO NOT open public issues for security issues
- Email: security@example.com (replace with actual)
- Include: description, impact, reproduction steps

**Security checklist:**
- [ ] No hardcoded credentials
- [ ] Input validation on all endpoints
- [ ] SQL injection prevention (use parameterized queries)
- [ ] JWT validation for protected endpoints
- [ ] Error messages don't leak sensitive info
- [ ] User authorization checks (users can only modify own articles)

## Code Review

Reviewers will check:
- Code quality and style
- Test coverage
- Documentation completeness
- Security considerations
- User Service integration handling
- Performance implications

## Questions?

- Open an issue for bugs/features
- Discussions for questions
- Check existing issues first

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
