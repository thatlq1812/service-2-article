package repository

import (
	pb "article-service/proto"
	"context"
)

// ArticleRepository define CRUD operations for articles
type ArticleRepository interface {
	// GetByID get article by ID
	GetByID(ctx context.Context, id int32) (*pb.Article, error)

	//Create new article
	Create(ctx context.Context, title, content string, userId int32) (*pb.Article, error)

	// Update article
	Update(ctx context.Context, id int32, title, content string) (*pb.Article, error)

	// Delete article
	Delete(ctx context.Context, id int32) error

	// ListByUser get article of 1 user (pagination)
	ListByUser(ctx context.Context, userId, limit, offset int32) ([]*pb.Article, int32, error)

	// ListAll
	ListAll(ctx context.Context, limit, offset int32) ([]*pb.Article, int32, error)
}
