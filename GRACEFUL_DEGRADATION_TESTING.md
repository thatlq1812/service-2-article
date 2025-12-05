# Graceful Degradation Testing Guide

## Objective
Test that Article Service continues to function when User Service is down or unavailable.

## Setup

### 1. Start All Services
```bash
cd /d/agrios
docker-compose up -d
sleep 15
bash scripts/init-services.sh
```

### 2. Create Test Data
```bash
# Create user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","email":"test@example.com","password":"pass123"}'

# Login
LOGIN_RESP=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}')

ACCESS_TOKEN=$(echo $LOGIN_RESP | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

# Create article
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Article","content":"This is test content"}'
```

---

## Test Scenarios

### Test 1: Normal Operation (Both Services Up)

**Action:**
```bash
curl http://localhost:8080/api/v1/articles/1 | jq '.'
```

**Expected Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "id": 1,
      "title": "Test Article",
      "content": "This is test content",
      "author": {
        "id": 1,
        "name": "Test User",
        "email": "test@example.com"
      },
      "created_at": "2025-12-05T...",
      "updated_at": "2025-12-05T..."
    }
  }
}
```

**✅ Pass Criteria:**
- Code: "000"
- Message: "success"
- Article data returned with full author information

---

### Test 2: User Service Down (Graceful Degradation)

**Action 1: Stop User Service**
```bash
docker-compose stop user-service
```

**Action 2: Get Article**
```bash
curl http://localhost:8080/api/v1/articles/1 | jq '.'
```

**Expected Response:**
```json
{
  "code": "000",
  "message": "success (author information unavailable)",
  "data": {
    "article": {
      "id": 1,
      "title": "Test Article",
      "content": "This is test content",
      "author": null,
      "created_at": "2025-12-05T...",
      "updated_at": "2025-12-05T..."
    }
  }
}
```

**✅ Pass Criteria:**
- Code: "000" (not "006")
- Message includes "author information unavailable"
- Article data returned
- Author field is null (not missing)
- Article Service did NOT crash

**Check Logs:**
```bash
docker-compose logs article-service | tail -20
```

**Expected Log Pattern:**
```
WARN: User Service unavailable (graceful degradation): article_id=1, user_id=1
WARN: Returned article without author info: article_id=1, user_id=1
```

---

### Test 3: List Articles with User Service Down

**Action: List Articles**
```bash
curl "http://localhost:8080/api/v1/articles?page=1&page_size=10" | jq '.'
```

**Expected Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "articles": [
      {
        "id": 1,
        "title": "Test Article",
        "content": "This is test content",
        "author": null,
        "created_at": "2025-12-05T..."
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 10,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

**✅ Pass Criteria:**
- Code: "000"
- Articles returned with author: null
- Pagination works correctly

**Check Logs:**
```bash
docker-compose logs article-service | grep "ListArticles"
```

**Expected Log Pattern:**
```
WARN: User Service unavailable (graceful degradation): article_id=1, user_id=1
WARN: 1/1 articles returned without author info due to User Service issues
```

---

### Test 4: Create Article with User Service Down

**Action: Try to Create Article**
```bash
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"New Article","content":"New content"}'
```

**Expected Response:**
```json
{
  "code": "006",
  "message": "user service is currently unavailable, please try again later"
}
```

**✅ Pass Criteria:**
- Code: "006" (Unavailable)
- Clear error message
- Article NOT created (this is expected - we need User Service for verification)

**Note:** This is intentional. CreateArticle requires User Service to verify user exists before creating article.

---

### Test 5: Service Recovery

**Action 1: Restart User Service**
```bash
docker-compose start user-service
sleep 5
```

**Action 2: Get Article Again**
```bash
curl http://localhost:8080/api/v1/articles/1 | jq '.'
```

**Expected Response:**
```json
{
  "code": "000",
  "message": "success",
  "data": {
    "article": {
      "id": 1,
      "title": "Test Article",
      "content": "This is test content",
      "author": {
        "id": 1,
        "name": "Test User",
        "email": "test@example.com"
      }
    }
  }
}
```

**✅ Pass Criteria:**
- Code: "000"
- Message: "success" (not "unavailable" anymore)
- Full author information returned
- Service automatically recovered

---

## Direct gRPC Testing (Optional)

### Test with grpcurl

**Normal Operation:**
```bash
grpcurl -plaintext -d '{"id":1}' localhost:50052 article.ArticleService.GetArticle
```

**With User Service Down:**
```bash
# Stop User Service
docker-compose stop user-service

# Test GetArticle
grpcurl -plaintext -d '{"id":1}' localhost:50052 article.ArticleService.GetArticle

# Check response has article but user is null
```

---

## Verification Checklist

After running all tests:

- [ ] Test 1 passed: Normal operation works with full author info
- [ ] Test 2 passed: GetArticle works when User Service down
- [ ] Test 3 passed: ListArticles works when User Service down
- [ ] Test 4 passed: CreateArticle fails gracefully (expected)
- [ ] Test 5 passed: Service recovers automatically
- [ ] Logs show WARN messages (not ERROR for graceful degradation)
- [ ] Article Service never crashed
- [ ] Response message indicates when author info unavailable

---

## Cleanup

```bash
# Start all services again
docker-compose start user-service

# Verify all services healthy
docker-compose ps

# Clean test data (optional)
bash scripts/clean-data.sh
docker-compose up -d
bash scripts/init-services.sh
```

---

## Summary

**What We Tested:**
1. ✅ Article retrieval continues when User Service down
2. ✅ Articles returned with `author: null` (graceful degradation)
3. ✅ Response message clearly indicates unavailable author info
4. ✅ Service automatically recovers when User Service back
5. ✅ Proper logging (WARN vs ERROR)

**Benefits:**
- Improved availability
- Better user experience
- Clear communication about partial data
- Automatic recovery without manual intervention
