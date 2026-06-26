package repository

import (
	"context"
	"database/sql"
	"scam-directory/internal/models"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO users (email, name, password_hash, google_id, avatar_url, role)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		user.Email, user.Name, user.PasswordHash, user.GoogleID, user.AvatarURL, user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = $1`, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepository) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE google_id = $1`, googleID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepository) UpdateGoogleID(ctx context.Context, userID, googleID, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET google_id = $1, avatar_url = $2, updated_at = NOW() WHERE id = $3`,
		googleID, avatarURL, userID,
	)
	return err
}
