package models

import "time"

type User struct {
	ID           string     `db:"id"            json:"id"`
	Email        string     `db:"email"         json:"email"`
	Name         string     `db:"name"          json:"name"`
	PasswordHash *string    `db:"password_hash" json:"-"`
	GoogleID     *string    `db:"google_id"     json:"-"`
	AvatarURL    *string    `db:"avatar_url"    json:"avatarUrl,omitempty"`
	Role         string     `db:"role"          json:"role"`
	CreatedAt    time.Time  `db:"created_at"    json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at"    json:"updatedAt"`
}
