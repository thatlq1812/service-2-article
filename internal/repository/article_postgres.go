package repository

import (
	"context"
	"fmt"
	"time"

	pb "github.com/thatlq1812/service-2-article/proto"

	"github.com/jackc/pgx/v5/pgxpool"
)

// articlePostgresRepo implement ArticleRepository with PostgreSQL
type articlePostgresRepo struct {
	db *pgxpool.Pool
}

// NewArticlePostgresRepository
func NewArticlePostgresRepository(db *pgxpool.Pool) ArticleRepository {
	return &articlePostgresRepo{db: db}
}

// GetByID
func (r *articlePostgresRepo) GetByID(ctx context.Context, id int32) (*pb.Article, error) {
	query := `
		SELECT id, title, content, user_id, created_at, updated_at
		FROM articles
		WHERE id = $1
	`
	var article pb.Article
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&article.Id,
		&article.Title,
		&article.Content,
		&article.UserId,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("query article failed: %w", err)
	}

	// Convert time to string
	article.CreatedAt = createdAt.Format(time.RFC3339)
	article.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &article, nil
}

// Create new article
func (r *articlePostgresRepo) Create(ctx context.Context, title, content string, userId int32) (*pb.Article, error) {
	query := `
		INSERT INTO articles (title, content, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, title, content, user_id, created_at, updated_at
	`
	var article pb.Article
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, query, title, content, userId).Scan(
		&article.Id,
		&article.Title,
		&article.Content,
		&article.UserId,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create article failed: %w", err)
	}

	article.CreatedAt = createdAt.Format(time.RFC3339)
	article.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &article, nil
}

// Update
func (r *articlePostgresRepo) Update(ctx context.Context, id int32, title, content string) (*pb.Article, error) {
	query := `
		UPDATE articles
		SET title = $1, content = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, title, content, user_id, created_at, updated_at
	`
	var article pb.Article
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, query, title, content, id).Scan(
		&article.Id,
		&article.Title,
		&article.Content,
		&article.UserId,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("update article failed: %w", err)
	}

	article.CreatedAt = createdAt.Format(time.RFC3339)
	article.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &article, nil
}

// Delete article
func (r *articlePostgresRepo) Delete(ctx context.Context, id int32) error {
	query := `DELETE FROM articles WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("Delete article failded: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("article with ID %d not found", id)
	}
	return nil
}

// ListByUser
func (r *articlePostgresRepo) ListByUser(ctx context.Context, userID, limit, offset int32) ([]*pb.Article, int32, error) {
	// Query
	query := `
		SELECT id, title, content, user_id, created_at, updated_at
		FROM articles
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query articles failed: %w", err)
	}
	defer rows.Close()

	var articles []*pb.Article

	for rows.Next() {
		var article pb.Article
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&article.Id,
			&article.Title,
			&article.Content,
			&article.UserId,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("scan article failed: %w", err)
		}

		article.CreatedAt = createdAt.Format(time.RFC3339)
		article.UpdatedAt = updatedAt.Format(time.RFC3339)
		articles = append(articles, &article)
	}

	// Count data
	countQuery := `SELECT COUNT(*) FROM articles WHERE user_id = $1`
	var total int32
	err = r.db.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count articles failed: %w", err)
	}
	return articles, total, nil
}

// ListAll articles
func (r *articlePostgresRepo) ListAll(ctx context.Context, limit, offset int32) ([]*pb.Article, int32, error) {
	query := `
		SELECT id, title, content, user_id, created_at, updated_at
		FROM articles
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query articles failed: %w", err)
	}
	defer rows.Close()

	var articles []*pb.Article

	for rows.Next() {
		var article pb.Article
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&article.Id,
			&article.Title,
			&article.Content,
			&article.UserId,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("scan article failed: %w", err)
		}

		article.CreatedAt = createdAt.Format(time.RFC3339)
		article.UpdatedAt = updatedAt.Format(time.RFC3339)
		articles = append(articles, &article)
	}

	// Count total articles
	countQuery := `SELECT COUNT(*) FROM articles`
	var total int32
	err = r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count articles failed: %w", err)
	}

	return articles, total, nil
}
