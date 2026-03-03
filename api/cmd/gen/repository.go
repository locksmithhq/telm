package main

// repository.go — data access layer.
// Each method opens a "db.*" span and simulates query latency.

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/locksmithhq/telm-go"
)

// ── User repository ────────────────────────────────────────────────────────

type userRepository struct{ serviceName string }

func (r *userRepository) FindByID(ctx context.Context, id string) (usr *user, err error) {
	ctx, end := telm.Start(ctx, "db.users.findByID", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"db.system":     "postgresql",
		"db.query.text": "SELECT * FROM users WHERE id = $1",
		"db.user.id":    id,
	})
	sleep(10, 40)
	telm.Event(ctx, "row.fetched", telm.F{"rows": 1})
	return &user{ID: id, Name: "User " + id, Email: id + "@example.com"}, nil
}

func (r *userRepository) CheckPermissions(ctx context.Context, userID, resource string) (ok bool, err error) {
	ctx, end := telm.Start(ctx, "db.permissions.check", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"db.system":     "postgresql",
		"db.query.text": "SELECT granted FROM permissions WHERE user_id=$1 AND resource=$2",
		"db.user.id":    userID,
		"db.resource":   resource,
	})
	sleep(5, 15)
	return true, nil
}

// ── Order repository ───────────────────────────────────────────────────────

type orderRepository struct{ serviceName string }

func (r *orderRepository) CheckInventory(ctx context.Context, productID string, qty int) (ok bool, err error) {
	ctx, end := telm.Start(ctx, "db.inventory.check", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"db.system":        "postgresql",
		"db.query.text":    "SELECT stock FROM inventory WHERE product_id = $1",
		"db.product.id":    productID,
		"db.requested.qty": qty,
	})
	sleep(8, 25)
	if rand.IntN(10) == 0 {
		telm.Event(ctx, "cache-hit")
		err = fmt.Errorf("insufficient stock for product %s", productID)
		return false, err
	}
	telm.Event(ctx, "cache-miss")
	telm.Count(ctx, "inventory.checks.total", 1, telm.F{"product": productID})
	return true, nil
}

func (r *orderRepository) Insert(ctx context.Context, o *order) (orderID string, err error) {
	ctx, end := telm.Start(ctx, "db.orders.insert", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"db.system":     "postgresql",
		"db.query.text": "INSERT INTO orders (user_id, product_id, amount) VALUES ($1,$2,$3) RETURNING id",
		"db.user.id":    o.UserID,
		"db.product.id": o.ProductID,
		"db.amount":     o.Amount,
	})
	sleep(15, 45)
	orderID = fmt.Sprintf("ord_%d", time.Now().UnixNano())
	telm.Event(ctx, "order.inserted", telm.F{"order.id": orderID})
	return orderID, nil
}

// ── Payment repository ─────────────────────────────────────────────────────

type paymentRepository struct{ serviceName string }

func (r *paymentRepository) GetCard(ctx context.Context, userID string) (c *card, err error) {
	ctx, end := telm.Start(ctx, "db.cards.getDefault", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"db.system":     "postgresql",
		"db.query.text": "SELECT * FROM payment_methods WHERE user_id=$1 AND default=true",
		"db.user.id":    userID,
	})
	sleep(8, 20)
	return &card{Token: "tok_" + userID, Last4: "4242", Brand: "visa"}, nil
}

func (r *paymentRepository) SaveTransaction(ctx context.Context, txID, orderID string, amount float64) (err error) {
	ctx, end := telm.Start(ctx, "db.transactions.save", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"db.system":     "postgresql",
		"db.query.text": "INSERT INTO transactions (id, order_id, amount, status) VALUES ($1,$2,$3,'completed')",
		"db.tx.id":      txID,
		"db.order.id":   orderID,
		"db.amount":     amount,
	})
	sleep(10, 30)
	return nil
}

// ── Models ─────────────────────────────────────────────────────────────────

type user struct {
	ID    string
	Name  string
	Email string
}

type order struct {
	ID        string
	UserID    string
	ProductID string
	Amount    float64
}

type card struct {
	Token string
	Last4 string
	Brand string
}

// ── Helpers ────────────────────────────────────────────────────────────────

func sleep(minMs, maxMs int) {
	time.Sleep(time.Duration(minMs+rand.IntN(maxMs-minMs)) * time.Millisecond)
}
