package postgres

import (
	"context"
	"encoding/json"
	"time"
)

type Dashboard struct {
	ID        string    `db:"id"          json:"id"`
	Name      string    `db:"name"        json:"name"`
	Panels    []Panel   `db:"-"           json:"panels"`
	PanelsRaw string    `db:"panels"      json:"-"`
	CreatedAt time.Time `db:"created_at"  json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at"  json:"updatedAt"`
}

type Panel struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Query string `json:"query"`
	Cols  int    `json:"cols"`
	Rows  int    `json:"rows"`
}

func (d *Dashboard) UnmarshalPanels() error {
	if d.PanelsRaw == "" || d.PanelsRaw == "null" {
		d.Panels = []Panel{}
		return nil
	}
	return json.Unmarshal([]byte(d.PanelsRaw), &d.Panels)
}

func (d *Dashboard) MarshalPanels() error {
	b, err := json.Marshal(d.Panels)
	if err != nil {
		return err
	}
	d.PanelsRaw = string(b)
	return nil
}

func (c *Client) ListDashboards(ctx context.Context) ([]Dashboard, error) {
	const q = `
		SELECT id, name, panels, created_at, updated_at
		FROM dashboards
		ORDER BY created_at DESC`

	var rows []Dashboard
	if err := c.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []Dashboard{}
	}
	for i := range rows {
		rows[i].UnmarshalPanels()
	}
	return rows, nil
}

func (c *Client) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	const q = `
		SELECT id, name, panels, created_at, updated_at
		FROM dashboards
		WHERE id = $1`

	var d Dashboard
	if err := c.db.GetContext(ctx, &d, q, id); err != nil {
		return nil, err
	}
	d.UnmarshalPanels()
	return &d, nil
}

func (c *Client) CreateDashboard(ctx context.Context, d *Dashboard) error {
	d.CreatedAt = time.Now()
	d.UpdatedAt = d.CreatedAt
	if err := d.MarshalPanels(); err != nil {
		return err
	}
	const q = `
		INSERT INTO dashboards (id, name, panels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := c.db.ExecContext(ctx, q, d.ID, d.Name, d.PanelsRaw, d.CreatedAt, d.UpdatedAt)
	return err
}

func (c *Client) UpdateDashboard(ctx context.Context, d *Dashboard) error {
	d.UpdatedAt = time.Now()
	if err := d.MarshalPanels(); err != nil {
		return err
	}
	const q = `
		UPDATE dashboards
		SET name = $2, panels = $3, updated_at = $4
		WHERE id = $1`
	result, err := c.db.ExecContext(ctx, q, d.ID, d.Name, d.PanelsRaw, d.UpdatedAt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (c *Client) DeleteDashboard(ctx context.Context, id string) error {
	const q = `DELETE FROM dashboards WHERE id = $1`
	result, err := c.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

var ErrNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "record not found" }
