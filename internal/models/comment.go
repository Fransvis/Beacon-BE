package models

import "time"

type Comment struct {
	ID          string     `db:"id"           json:"id"`
	ScamID      string     `db:"scam_id"      json:"scamId"`
	UserID      *string    `db:"user_id"      json:"userId,omitempty"`
	AuthorName  *string    `db:"author_name"  json:"authorName,omitempty"`
	Content     string     `db:"content"      json:"content"`
	IsAnonymous bool       `db:"is_anonymous" json:"isAnonymous"`
	CreatedAt   time.Time  `db:"created_at"   json:"createdAt"`
}
