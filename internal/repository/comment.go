package repository

import (
	"context"

	"scam-directory/internal/models"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type CommentRepository struct {
	db *sqlx.DB
}

func NewCommentRepository(db *sqlx.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, comment *models.Comment) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO comments (scam_id, user_id, author_name, content, is_anonymous)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		comment.ScamID, comment.UserID, comment.AuthorName, comment.Content, comment.IsAnonymous,
	).Scan(&comment.ID, &comment.CreatedAt)
}

func (r *CommentRepository) GetByScamID(ctx context.Context, scamID uuid.UUID) ([]models.Comment, error) {
	comments := make([]models.Comment, 0)
	query := `
		SELECT c.id, c.scam_id, c.user_id,
		       COALESCE(u.name, c.author_name) AS author_name,
		       c.content, c.is_anonymous, c.created_at
		FROM comments c
		LEFT JOIN users u ON u.id = c.user_id
		WHERE c.scam_id = $1
		ORDER BY c.created_at DESC`
	return comments, r.db.SelectContext(ctx, &comments, query, scamID)
}

func (r *CommentRepository) CountByScamID(ctx context.Context, scamID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM comments WHERE scam_id = $1`, scamID)
	return count, err
}
