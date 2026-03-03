package main

// usecase.go — business layer.
// Orchestrates repositories and gateways; each use case opens its own spans.

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/locksmithhq/telm-go"
)

// ── User usecase ───────────────────────────────────────────────────────────

type userUsecase struct {
	repo *userRepository
}

func (u *userUsecase) ValidateUser(ctx context.Context, userID string) (usr *user, err error) {
	ctx, end := telm.Start(ctx, "usecase.user.validate", telm.Internal())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{"user.id": userID})

	usr, err = u.repo.FindByID(ctx, userID)
	if err != nil {
		telm.Error(ctx, "user not found", err, telm.F{"user_id": userID})
		return nil, fmt.Errorf("user not found: %w", err)
	}

	allowed, err := u.repo.CheckPermissions(ctx, userID, "orders:create")
	if err != nil || !allowed {
		telm.Warn(ctx, "permission denied", telm.F{"user_id": userID, "resource": "orders:create"})
		err = fmt.Errorf("permission denied for user %s", userID)
		return nil, err
	}

	telm.Info(ctx, "user validated", telm.F{"user_id": userID, "email": usr.Email})
	telm.Count(ctx, "users.validated", 1)
	return usr, nil
}

// ── Order usecase ──────────────────────────────────────────────────────────

type orderUsecase struct {
	repo    *orderRepository
	payment *paymentUsecase
}

func (u *orderUsecase) CreateOrder(ctx context.Context, usr *user, productID string, amount float64) (orderID string, err error) {
	ctx, end := telm.Start(ctx, "usecase.order.create", telm.Internal())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"user.id":      usr.ID,
		"product.id":   productID,
		"order.amount": amount,
	})

	qty := rand.IntN(5) + 1
	ok, err := u.repo.CheckInventory(ctx, productID, qty)
	if err != nil || !ok {
		telm.Warn(ctx, "insufficient stock", telm.F{"product_id": productID, "qty": qty})
		return "", fmt.Errorf("insufficient stock for product %s", productID)
	}

	o := &order{UserID: usr.ID, ProductID: productID, Amount: amount}
	orderID, err = u.repo.Insert(ctx, o)
	if err != nil {
		telm.Error(ctx, "order insert failed", err)
		return "", err
	}
	o.ID = orderID

	txID, err := u.payment.ChargeCard(ctx, usr.ID, orderID, amount)
	if err != nil {
		telm.Error(ctx, "payment failed", err, telm.F{"order_id": orderID})
		return "", fmt.Errorf("payment failed: %w", err)
	}

	telm.Info(ctx, "order created", telm.F{"order_id": orderID, "tx_id": txID, "amount": amount})
	telm.Count(ctx, "orders.created", 1, telm.F{"product": productID})
	telm.Record(ctx, "orders.amount", amount, telm.F{"product": productID})
	return orderID, nil
}

// ── Payment usecase ────────────────────────────────────────────────────────

type paymentUsecase struct {
	repo   *paymentRepository
	stripe *stripeGateway
}

func (u *paymentUsecase) ChargeCard(ctx context.Context, userID, orderID string, amount float64) (string, error) {
	txID, err := u.processCharge(ctx, userID, orderID, amount)
	return txID, err
}

func (u *paymentUsecase) processCharge(ctx context.Context, userID, orderID string, amount float64) (txID string, err error) {
	ctx, end := telm.Start(ctx, "usecase.payment.charge", telm.Server())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"user.id":        userID,
		"order.id":       orderID,
		"payment.amount": amount,
	})

	c, err := u.repo.GetCard(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("card not found: %w", err)
	}
	telm.Debug(ctx, "card retrieved", telm.F{"brand": c.Brand, "last4": c.Last4})

	txID, err = u.stripe.Charge(ctx, c.Token, amount, "BRL")
	if err != nil {
		telm.Count(ctx, "payments.failed", 1, telm.F{"reason": "stripe_decline"})
		return "", err
	}

	if sErr := u.repo.SaveTransaction(ctx, txID, orderID, amount); sErr != nil {
		telm.Warn(ctx, "transaction save failed", telm.F{"tx_id": txID})
	}

	telm.Info(ctx, "payment processed", telm.F{"tx_id": txID, "amount": amount, "brand": c.Brand})
	telm.Count(ctx, "payments.processed", 1, telm.F{"brand": c.Brand})
	telm.Record(ctx, "payments.amount", amount)
	return txID, nil
}

// ── Notification usecase ───────────────────────────────────────────────────

type notificationUsecase struct {
	email *emailGateway
}

func (u *notificationUsecase) Notify(ctx context.Context, to, subject, template string) error {
	err := u.send(ctx, to, subject, template)
	return err
}

func (u *notificationUsecase) send(ctx context.Context, to, subject, template string) (err error) {
	ctx, end := telm.Start(ctx, "usecase.notification.send", telm.Server())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"notification.to":       to,
		"notification.template": template,
	})

	if sendErr := u.email.Send(ctx, to, subject, template); sendErr != nil {
		telm.Warn(ctx, "email delivery failed", telm.F{"to": to, "template": template, "error": sendErr.Error()})
		return nil // non-critical
	}

	telm.Info(ctx, "notification sent", telm.F{"to": to, "template": template})
	telm.Count(ctx, "notifications.sent", 1, telm.F{"template": template})
	return nil
}
