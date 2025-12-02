package server

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"article-service/internal/client"
	"article-service/internal/repository"
	pb "article-service/proto"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
	errNoRows       = "no rows"
)

// articleServiceServer implements ArticleServiceServer
type articleServiceServer struct {
	pb.UnimplementedArticleServiceServer
	repo       repository.ArticleRepository
	userClient *client.UserClient
}

// NewArticleServiceServer creates new server instance
func NewArticleServiceServer(repo repository.ArticleRepository, userClient *client.UserClient) pb.ArticleServiceServer {
	return &articleServiceServer{
		repo:       repo,
		userClient: userClient,
	}
}

// GetArticle retrieves an article with user information
// This is a convenience method that delegates to GetArticleWithUser
func (s *articleServiceServer) GetArticle(ctx context.Context, req *pb.GetArticleRequest) (*pb.ArticleWithUser, error) {
	return s.GetArticleWithUser(ctx, req)
}

// GetArticleWithUser retrieves an article with associated user information via inter-service communication
// If the user is not found or deleted, returns the article with a nil user
func (s *articleServiceServer) GetArticleWithUser(ctx context.Context, req *pb.GetArticleRequest) (*pb.ArticleWithUser, error) {
	// Validate input
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "article ID must be positive")
	}

	// 1. Retrieve article from database
	article, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), errNoRows) {
			return nil, status.Errorf(codes.NotFound, "article with ID %d not found", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to get article: %v", err)
	}

	// 2. Fetch user information from User Service (inter-service communication)
	user, err := s.userClient.GetUser(ctx, article.UserId)
	if err != nil {
		// If user not found, still return article with nil user (user may be deleted)
		if status.Code(err) == codes.NotFound {
			return &pb.ArticleWithUser{
				Article: article,
				User:    nil,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to get user from user service: %v", err)
	}

	// 3. Return combined article and user data
	return &pb.ArticleWithUser{
		Article: article,
		User:    user,
	}, nil
}

// CreateArticle creates a new article after verifying the user exists
func (s *articleServiceServer) CreateArticle(ctx context.Context, req *pb.CreateArticleRequest) (*pb.Article, error) {
	// Validate input
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user ID must be positive")
	}

	// Verify user exists by calling User Service
	_, err := s.userClient.GetUser(ctx, req.UserId)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Errorf(codes.InvalidArgument, "user with ID %d not found", req.UserId)
		}
		return nil, status.Errorf(codes.Internal, "failed to verify user: %v", err)
	}

	// Create article in database
	article, err := s.repo.Create(ctx, req.Title, req.Content, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create article: %v", err)
	}

	return article, nil
}

// UpdateArticle updates an article's title and/or content
// Partial updates are supported - omitted fields retain their existing values
func (s *articleServiceServer) UpdateArticle(ctx context.Context, req *pb.UpdateArticleRequest) (*pb.Article, error) {
	// Validate input
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "article ID must be positive")
	}
	if req.Title == "" && req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "at least title or content must be provided")
	}

	// Check if article exists and get current values
	existing, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), errNoRows) {
			return nil, status.Errorf(codes.NotFound, "article with ID %d not found", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to check article: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to update article: %v", err)
	}

	return article, nil
}

// DeleteArticle deletes an article and returns the deleted article data
func (s *articleServiceServer) DeleteArticle(ctx context.Context, req *pb.DeleteArticleRequest) (*pb.Article, error) {
	// Validate input
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "article ID must be positive")
	}

	// Get article before deletion to return it in response
	article, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), errNoRows) {
			return nil, status.Errorf(codes.NotFound, "article with ID %d not found", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to get article: %v", err)
	}

	// Delete article from database
	err = s.repo.Delete(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete article: %v", err)
	}

	return article, nil
}

// ListArticles retrieves a paginated list of articles with user information
// Supports filtering by user ID. Fetches user data for each article via inter-service communication.
func (s *articleServiceServer) ListArticles(ctx context.Context, req *pb.ListArticlesRequest) (*pb.ListArticlesResponse, error) {
	// Validate and normalize pagination parameters
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	pageNumber := req.PageNumber
	if pageNumber < 0 {
		pageNumber = 0
	}

	// Calculate offset for pagination
	offset := pageNumber * pageSize

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
		return nil, status.Errorf(codes.Internal, "failed to list articles: %v", err)
	}

	// Enrich articles with user information from User Service
	articlesWithUser := make([]*pb.ArticleWithUser, 0, len(articles))
	for _, article := range articles {
		user, err := s.userClient.GetUser(ctx, article.UserId)
		if err != nil {
			// If user not found, include article with nil user (user may be deleted)
			user = nil
		}
		articlesWithUser = append(articlesWithUser, &pb.ArticleWithUser{
			Article: article,
			User:    user,
		})
	}

	// Calculate total pages
	totalPages := (total + pageSize - 1) / pageSize

	return &pb.ListArticlesResponse{
		Articles:   articlesWithUser,
		Total:      total,
		Page:       pageNumber,
		TotalPages: totalPages,
	}, nil
}
