package server

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"article-service/internal/client"
	"article-service/internal/repository"
	pb "article-service/proto"
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

// GetArticle implements GetArticle RPC
// Returns article with user info (same as GetArticleWithUser)
func (s *articleServiceServer) GetArticle(ctx context.Context, req *pb.GetArticleRequest) (*pb.ArticleWithUser, error) {
	// Validate input
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "article ID must be positive")
	}

	// Get article from database
	article, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("article with ID %d not found", req.Id))
		}
		return nil, status.Error(codes.Internal, "failed to get article")
	}

	// Call Service-1 to get user info
	user, err := s.userClient.GetUser(ctx, article.UserId)
	if err != nil {
		// If user not found, still return article but with nil user
		if status.Code(err) == codes.NotFound {
			return &pb.ArticleWithUser{
				Article: article,
				User:    nil,
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get user from service-1")
	}

	return &pb.ArticleWithUser{
		Article: article,
		User:    user,
	}, nil
}

// GetArticleWithUser implements GetArticleWithUser RPC
// This demonstrates inter-service communication
func (s *articleServiceServer) GetArticleWithUser(ctx context.Context, req *pb.GetArticleRequest) (*pb.ArticleWithUser, error) {
	// Validate input
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "article ID must be positive")
	}

	// 1. Get article from database
	article, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("article with ID %d not found", req.Id))
		}
		return nil, status.Error(codes.Internal, "failed to get article")
	}

	// 2. Call Service-1 to get user info (inter-service communication)
	user, err := s.userClient.GetUser(ctx, article.UserId)
	if err != nil {
		// If user not found, still return article but with nil user
		if status.Code(err) == codes.NotFound {
			return &pb.ArticleWithUser{
				Article: article,
				User:    nil, // User deleted or not found
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get user from service-1")
	}

	// 3. Combine article + user data
	return &pb.ArticleWithUser{
		Article: article,
		User:    user,
	}, nil
}

// CreateArticle implements CreateArticle RPC
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

	// Verify user exists by calling Service-1
	_, err := s.userClient.GetUser(ctx, req.UserId)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("user with ID %d not found", req.UserId))
		}
		return nil, status.Error(codes.Internal, "failed to verify user")
	}

	// Create article in database
	article, err := s.repo.Create(ctx, req.Title, req.Content, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create article")
	}

	return article, nil
}

// UpdateArticle implements UpdateArticle RPC
func (s *articleServiceServer) UpdateArticle(ctx context.Context, req *pb.UpdateArticleRequest) (*pb.Article, error) {
	// Validate input
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "article ID must be positive")
	}
	if req.Title == "" && req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "at least title or content must be provided")
	}

	// Check if article exists
	existing, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("article with ID %d not found", req.Id))
		}
		return nil, status.Error(codes.Internal, "failed to check article")
	}

	// Use existing values if not provided
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
		return nil, status.Error(codes.Internal, "failed to update article")
	}

	return article, nil
}

// DeleteArticle implements DeleteArticle RPC
func (s *articleServiceServer) DeleteArticle(ctx context.Context, req *pb.DeleteArticleRequest) (*pb.Article, error) {
	// Validate input
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "article ID must be positive")
	}

	// Get article before delete (to return it)
	article, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("article with ID %d not found", req.Id))
		}
		return nil, status.Error(codes.Internal, "failed to get article")
	}

	// Delete article
	err = s.repo.Delete(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete article")
	}

	return article, nil
}

// ListArticles implements ListArticles RPC
func (s *articleServiceServer) ListArticles(ctx context.Context, req *pb.ListArticlesRequest) (*pb.ListArticlesResponse, error) {
	// Validate pagination
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10 // default
	}
	if pageSize > 100 {
		pageSize = 100 // max limit
	}

	pageNumber := req.PageNumber
	if pageNumber < 0 {
		pageNumber = 0
	}

	// Calculate offset
	offset := pageNumber * pageSize

	// Get articles based on filter
	var articles []*pb.Article
	var total int32
	var err error

	if req.UserId > 0 {
		// Filter by user
		articles, total, err = s.repo.ListByUser(ctx, req.UserId, pageSize, offset)
	} else {
		// List all
		articles, total, err = s.repo.ListAll(ctx, pageSize, offset)
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list articles")
	}

	// Convert Article to ArticleWithUser (fetch user info for each)
	articlesWithUser := make([]*pb.ArticleWithUser, 0, len(articles))
	for _, article := range articles {
		user, err := s.userClient.GetUser(ctx, article.UserId)
		if err != nil {
			// If user not found, still include article with nil user
			user = nil
		}
		articlesWithUser = append(articlesWithUser, &pb.ArticleWithUser{
			Article: article,
			User:    user,
		})
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize

	return &pb.ListArticlesResponse{
		Articles:   articlesWithUser,
		Total:      total,
		Page:       pageNumber,
		TotalPages: totalPages,
	}, nil
}
