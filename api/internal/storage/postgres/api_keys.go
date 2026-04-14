package postgres

import (
	"context"
	"time"
)

type APIKey struct {
	ID          int64      `db:"id"`
	Name        string     `db:"name"`
	KeyHash     string     `db:"key_hash"`
	CreatedAt   time.Time  `db:"created_at"`
	LastUsedAt  *time.Time `db:"last_used_at"`
}

func (c *Client) CreateAPIKey(ctx context.Context, name, keyHash string) (int64, error) {
	var id int64
	err := c.db.QueryRowContext(ctx,
		`INSERT INTO api_keys (name, key_hash) VALUES ($1, $2) RETURNING id`,
		name, keyHash,
	).Scan(&id)
	return id, err
}

func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var keys []APIKey
	err := c.db.SelectContext(ctx, &keys,
		`SELECT id, name, key_hash, created_at, last_used_at FROM api_keys ORDER BY created_at DESC`)
	if keys == nil {
		keys = []APIKey{}
	}
	return keys, err
}

func (c *Client) RevokeAPIKey(ctx context.Context, id int64) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	return err
}

func (c *Client) FindAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	var k APIKey
	err := c.db.GetContext(ctx, &k,
		`SELECT id, name, key_hash, created_at, last_used_at FROM api_keys WHERE key_hash = $1`,
		keyHash,
	)
	if err != nil {
		return nil, err
	}
	go c.db.ExecContext(context.Background(), //nolint:errcheck
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, k.ID)
	return &k, nil
}
