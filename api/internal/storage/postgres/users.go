package postgres

import (
	"context"
	"errors"
	"time"
)

type User struct {
	ID        int64     `db:"id"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	CreatedAt time.Time `db:"created_at"`
}

var ErrUserNotFound = errors.New("user not found")

func (c *Client) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	if err := c.db.GetContext(ctx, &u, `SELECT id, email, password, created_at FROM users WHERE email = $1`, email); err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// UpsertAdminUser inserts if not exists — ON CONFLICT DO NOTHING preserva senha customizada ao reiniciar.
func (c *Client) UpsertAdminUser(ctx context.Context, email, hashedPassword string) error {
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO users (email, password) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING`,
		email, hashedPassword,
	)
	return err
}
