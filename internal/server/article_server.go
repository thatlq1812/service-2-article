package server

import (
	"context"
	"fmt"
	"log"
	"strings"

	userpb "service-1-user/proto"
	"service-2-article/internal/auth"
	"service-2-article/internal/client"
	"service-2-article/internal/repository"
	"service-2-article/internal/response"
	pb "service-2-article/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
	errNoRows       = "no rows"
)

// convertUser converts User Service User to Article Service User proto type
func convertUser(userServiceUser *userpb.User) *pb.User {
	if userServiceUser == nil {
		return nil
	}
	return &pb.User{
		Id:        userServiceUser.Id,
		Name:      userServiceUser.Name,
		Email:     userServiceUser.Email,
		CreatedAt: userServiceUser.CreatedAt,
		UpdatedAt: userServiceUser.UpdatedAt,
	}
}

type ArticleServer struct {
	pb.UnimplementedArticleServiceServer
	repo       repository.ArticleRepository
	userClient *client.UserClient
	redis      auth.TokenBlacklistChecker
	jwtSecret  string
}

func NewArticleServer(repo repository.ArticleRepository, userClient *client.UserClient, redis auth.TokenBlacklistChecker, jwtSecret string) *ArticleServer {
	return &ArticleServer{
		repo:       repo,
		userClient: userClient,
		redis:      redis,
		jwtSecret:  jwtSecret,
	}
}

func (s *ArticleServer) CreateArticle(ctx context.Context, req *pb.CreateArticleRequest) (*pb.CreateArticleResponse, error) {
	// Validate authentication (JWT + Redis blacklist check)
	userID, err := auth.GetUserIDFromContextWithBlacklist(ctx, s.jwtSecret, s.redis)
	if err != nil {
		if err == auth.ErrTokenBlacklisted {
			log.Printf("[CreateArticle] Token has been revoked (logged out)")
			return response.CreateArticleError(codes.Unauthenticated, "token has been revoked"), nil
		}
		log.Printf("[CreateArticle] Authentication failed: %v", err)
		return response.CreateArticleError(codes.Unauthenticated, "authentication required"), nil
	}

	// Validate input
	if req.Title == "" {
		log.Printf("[CreateArticle] Invalid argument: title is empty")
		return response.CreateArticleError(codes.InvalidArgument, "title is required"), nil
	}
	if req.Content == "" {
		log.Printf("[CreateArticle] Invalid argument: content is empty")
		return response.CreateArticleError(codes.InvalidArgument, "content is required"), nil
	}

	// Verify user exists by calling User Service
	log.Printf("[CreateArticle] Verifying user exists: user_id=%d", userID)
	_, err = s.userClient.GetUser(ctx, int32(userID))
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.NotFound:
			log.Printf("[CreateArticle] User not found: user_id=%d", userID)
			return response.CreateArticleError(codes.InvalidArgument, fmt.Sprintf("user with ID %d not found", userID)), nil
		case codes.Unavailable:
			log.Printf("[CreateArticle] User service unavailable: user_id=%d", userID)
			return response.CreateArticleError(codes.Unavailable, "user service is currently unavailable, please try again later"), nil
		case codes.DeadlineExceeded:
			log.Printf("[CreateArticle] User service timeout: user_id=%d", userID)
			return response.CreateArticleError(codes.DeadlineExceeded, "request timeout while verifying user"), nil
		default:
			log.Printf("[CreateArticle] Failed to verify user: user_id=%d, error=%v", userID, err)
			return response.CreateArticleError(codes.Internal, "failed to verify user"), nil
		}
	}

	// Create article in database with authenticated user ID
	article, err := s.repo.Create(ctx, req.Title, req.Content, int32(userID))
	if err != nil {
		log.Printf("[CreateArticle] Database error: user_id=%d, error=%v", userID, err)
		return response.CreateArticleError(codes.Internal, "failed to create article"), nil
	}

	log.Printf("[CreateArticle] Success: article_id=%d, user_id=%d", article.Id, userID)
	return response.CreateArticleSuccess(article), nil
}

// GetArticle retrieves an article with user information
// This is a convenience method that delegates to GetArticleWithUser
func (s *ArticleServer) GetArticle(ctx context.Context, req *pb.GetArticleRequest) (*pb.GetArticleResponse, error) {
	// Validate input
	if req.Id <= 0 {
		log.Printf("[GetArticle] Invalid argument: article_id=%d", req.Id)
		return response.GetArticleError(codes.InvalidArgument, "article ID must be positive"), nil
	}

	// Get article with user
	article, err := s.GetArticleWithUser(ctx, req)
	if err != nil {
		// GetArticleWithUser already returns wrapped errors in ArticleWithUser
		// We need to convert to GetArticleResponse error
		st := status.Convert(err)
		return response.GetArticleError(st.Code(), st.Message()), nil
	}
	return response.GetArticleSuccess(article), nil
}

// GetArticleWithUser retrieves an article with associated user information via inter-service communication
// If the user is not found or deleted, returns the article with a nil user
func (s *ArticleServer) GetArticleWithUser(ctx context.Context, req *pb.GetArticleRequest) (*pb.ArticleWithUser, error) {
	// Validate input
	if req.Id <= 0 {
		log.Printf("[GetArticleWithUser] Invalid argument: article_id=%d", req.Id)
		return nil, response.GRPCError(codes.InvalidArgument, "Article ID must be positive. Provide a valid ID greater than 0.")
	}

	// 1. Retrieve article from database
	article, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), errNoRows) {
			log.Printf("[GetArticleWithUser] Article not found: article_id=%d", req.Id)
			return nil, response.GRPCError(codes.NotFound, "Article not found. Verify the article ID exists.")
		}
		log.Printf("[GetArticleWithUser] Database error: article_id=%d, error=%v", req.Id, err)
		return nil, response.GRPCError(codes.Internal, "Failed to get article. Contact support if the issue persists.")
	}

	// 2. Fetch user information from User Service (inter-service communication)
	log.Printf("[GetArticleWithUser] Fetching user info: article_id=%d, user_id=%d", article.Id, article.UserId)
	userServiceUser, err := s.userClient.GetUser(ctx, article.UserId)
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.NotFound:
			log.Printf("[GetArticleWithUser] User not found (graceful degradation): article_id=%d, user_id=%d", article.Id, article.UserId)
			return &pb.ArticleWithUser{
				Article: article,
				User:    nil,
			}, nil
		case codes.Unavailable, codes.DeadlineExceeded:
			log.Printf("[GetArticleWithUser] User service unavailable (graceful degradation): article_id=%d, user_id=%d, code=%s", article.Id, article.UserId, st.Code())
			return &pb.ArticleWithUser{
				Article: article,
				User:    nil,
			}, nil
		default:
			log.Printf("[GetArticleWithUser] User service error: article_id=%d, user_id=%d, error=%v", article.Id, article.UserId, err)
			return nil, response.GRPCError(codes.Internal, "Failed to get user from user service. Contact support if the issue persists.")
		}
	}

	// 3. Convert and return combined article and user data
	log.Printf("[GetArticleWithUser] Success: article_id=%d, user_id=%d, user_email=%s", article.Id, userServiceUser.Id, userServiceUser.Email)
	return &pb.ArticleWithUser{
		Article: article,
		User:    convertUser(userServiceUser),
	}, nil
}

// CreateArticleOld creates a new article after verifying the user exists (DEPRECATED - use CreateArticle with auth)
func (s *ArticleServer) CreateArticleOld(ctx context.Context, req *pb.CreateArticleRequest) (*pb.Article, error) {
	// Validate input
	if req.Title == "" {
		log.Printf("[CreateArticle] Invalid argument: title is empty")
		return nil, response.GRPCError(codes.InvalidArgument, "Title is required. Provide a valid title.")
	}
	if req.Content == "" {
		log.Printf("[CreateArticle] Invalid argument: content is empty")
		return nil, response.GRPCError(codes.InvalidArgument, "Content is required. Provide valid content.")
	}
	if req.UserId <= 0 {
		log.Printf("[CreateArticle] Invalid argument: user_id=%d", req.UserId)
		return nil, response.GRPCError(codes.InvalidArgument, "User ID must be positive. Provide a valid ID greater than 0.")
	}

	// Verify user exists by calling User Service with retry
	log.Printf("[CreateArticle] Verifying user exists: user_id=%d", req.UserId)
	_, err := s.userClient.GetUserWithRetry(ctx, req.UserId)
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.NotFound:
			log.Printf("[CreateArticle] User not found: user_id=%d", req.UserId)
			return nil, response.GRPCError(codes.InvalidArgument, fmt.Sprintf("User with ID %d not found. Verify the user ID exists.", req.UserId))
		case codes.Unavailable:
			log.Printf("[CreateArticle] User service unavailable: user_id=%d", req.UserId)
			return nil, response.GRPCError(codes.Unavailable, "User service is currently unavailable. Please try again later.")
		case codes.DeadlineExceeded:
			log.Printf("[CreateArticle] User service timeout: user_id=%d", req.UserId)
			return nil, response.GRPCError(codes.DeadlineExceeded, "Request timeout while verifying user. Please try again.")
		default:
			log.Printf("[CreateArticle] Failed to verify user: user_id=%d, error=%v", req.UserId, err)
			return nil, response.GRPCError(codes.Internal, fmt.Sprintf("Failed to verify user: %v. Contact support if the issue persists.", err))
		}
	}

	// Create article in database
	article, err := s.repo.Create(ctx, req.Title, req.Content, req.UserId)
	if err != nil {
		log.Printf("[CreateArticle] Database error: user_id=%d, error=%v", req.UserId, err)
		return nil, response.GRPCError(codes.Internal, fmt.Sprintf("Failed to create article: %v. Contact support if the issue persists.", err))
	}

	log.Printf("[CreateArticle] Success: article_id=%d, user_id=%d", article.Id, req.UserId)
	return article, nil
}

// UpdateArticle updates an article's title and/or content
// Partial updates are supported - omitted fields retain their existing values
func (s *ArticleServer) UpdateArticle(ctx context.Context, req *pb.UpdateArticleRequest) (*pb.UpdateArticleResponse, error) {
	// Validate input
	if req.Id <= 0 {
		return response.UpdateArticleError(codes.InvalidArgument, "article ID must be positive"), nil
	}
	if req.Title == "" && req.Content == "" {
		return response.UpdateArticleError(codes.InvalidArgument, "at least title or content must be provided"), nil
	}

	// Check if article exists and get current values
	existing, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), errNoRows) {
			return response.UpdateArticleError(codes.NotFound, fmt.Sprintf("article with ID %d not found", req.Id)), nil
		}
		return response.UpdateArticleError(codes.Internal, "failed to check article"), nil
	}

	// Use existing values for omitted fields
	title := req.Title
	if title == "" {
		title = existing.Title
	}
	content := req.Content
	if content == "" {
		content = existing.Content
	}

	// Update article
	article, err := s.repo.Update(ctx, req.Id, title, content)
	if err != nil {
		return response.UpdateArticleError(codes.Internal, "failed to update article"), nil
	}

	return response.UpdateArticleSuccess(article), nil
}

// DeleteArticle deletes an article and returns the deleted article data
func (s *ArticleServer) DeleteArticle(ctx context.Context, req *pb.DeleteArticleRequest) (*pb.DeleteArticleResponse, error) {
	// Validate input
	if req.Id <= 0 {
		return response.DeleteArticleError(codes.InvalidArgument, "article ID must be positive"), nil
	}

	// Check if article exists before deletion
	_, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), errNoRows) {
			return response.DeleteArticleError(codes.NotFound, "article not found"), nil
		}
		return response.DeleteArticleError(codes.Internal, "failed to check article"), nil
	}

	// Delete article from database
	err = s.repo.Delete(ctx, req.Id)
	if err != nil {
		return response.DeleteArticleError(codes.Internal, "failed to delete article"), nil
	}

	return response.DeleteArticleSuccess(), nil
}

// ListArticles retrieves a paginated list of articles with user information
// Supports filtering by user ID. Fetches user data for each article via inter-service communication.
func (s *ArticleServer) ListArticles(ctx context.Context, req *pb.ListArticlesRequest) (*pb.ListArticlesResponse, error) {
	// Validate and normalize pagination parameters
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	pageNumber := req.PageNumber
	if pageNumber < 1 {
		pageNumber = 1
	}

	// Calculate offset for pagination (page starts from 1)
	offset := (pageNumber - 1) * pageSize

	// Retrieve articles based on filter
	var articles []*pb.Article
	var total int32
	var err error

	if req.UserId > 0 {
		// Filter by specific user
		articles, total, err = s.repo.ListByUser(ctx, req.UserId, pageSize, offset)
	} else {
		// List all articles
		articles, total, err = s.repo.ListAll(ctx, pageSize, offset)
	}

	if err != nil {
		return response.ListArticlesError(codes.Internal, "failed to list articles"), nil
	}

	// Enrich articles with user information from User Service
	log.Printf("[ListArticles] Fetching user info for %d articles", len(articles))
	articlesWithUser := make([]*pb.ArticleWithUser, 0, len(articles))
	for _, article := range articles {
		userServiceUser, err := s.userClient.GetUser(ctx, article.UserId)
		if err != nil {
			// If user not found or service unavailable, include article with nil user (graceful degradation)
			log.Printf("[ListArticles] Failed to get user (graceful degradation): article_id=%d, user_id=%d, error=%v", article.Id, article.UserId, err)
			userServiceUser = nil
		}
		articlesWithUser = append(articlesWithUser, &pb.ArticleWithUser{
			Article: article,
			User:    convertUser(userServiceUser),
		})
	}

	log.Printf("[ListArticles] Success: returned=%d, total=%d, page=%d", len(articlesWithUser), total, pageNumber)

	// Calculate total pages
	totalPages := (total + pageSize - 1) / pageSize

	return response.ListArticlesSuccess(articlesWithUser, total, pageNumber, totalPages), nil
}
