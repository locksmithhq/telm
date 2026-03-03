package main

// gateway.go — calls to external services (Stripe, email, etc.)
// Uses telm.InjectHeaders / telm.ExtractHeaders to propagate the trace context,
// simulating a real HTTP call between microservices.

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/locksmithhq/telm-go"
)

// ── Stripe gateway ─────────────────────────────────────────────────────────

type stripeGateway struct{}

func (g *stripeGateway) Charge(ctx context.Context, token string, amount float64, currency string) (txID string, err error) {
	ctx, end := telm.Start(ctx, "http.stripe.charge", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"http.request.method": "POST",
		"http.url":            "https://api.stripe.com/v1/charges",
		"net.peer.name":       "api.stripe.com",
		"peer.service":        "stripe",
		"stripe.token":        token[:4] + "***",
		"stripe.amount":       amount,
		"stripe.currency":     currency,
	})
	sleep(40, 120)

	if rand.IntN(15) == 0 {
		err = fmt.Errorf("stripe: card declined (insufficient funds)")
		return "", err
	}

	txID = fmt.Sprintf("ch_%d", time.Now().UnixNano())
	telm.Event(ctx, "stripe.charge.succeeded", telm.F{"charge.id": txID})
	telm.Count(ctx, "stripe.charges.total", 1, telm.F{"currency": currency, "status": "success"})
	telm.Record(ctx, "stripe.charge.amount", amount, telm.F{"currency": currency})
	return txID, nil
}

// ── Email gateway ──────────────────────────────────────────────────────────

type emailGateway struct{}

func (g *emailGateway) Send(ctx context.Context, to, subject, template string) (err error) {
	ctx, end := telm.Start(ctx, "http.email.send", telm.Client())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"http.request.method": "POST",
		"http.url":            "https://api.sendgrid.com/v3/mail/send",
		"peer.service":        "sendgrid",
		"email.to":            to,
		"email.subject":       subject,
		"email.template":      template,
	})
	sleep(20, 60)

	if rand.IntN(20) == 0 {
		err = fmt.Errorf("email: sendgrid rate limit exceeded")
		return err
	}

	telm.Event(ctx, "email.queued")
	telm.Count(ctx, "emails.sent", 1, telm.F{"template": template})
	return nil
}

// ── Inter-service HTTP simulation ──────────────────────────────────────────
// serviceStart simulates an HTTP call between microservices:
//  1. Opens a CLIENT span in the calling service
//  2. Injects the W3C trace context into simulated "headers"
//  3. Extracts the context in the receiving service (new "remote" context)
//
// The caller receives remoteCtx (to use in the receiving service's handler)
// and end (to close the CLIENT span when the call completes).
//
// Usage:
//
//	remoteCtx, end := serviceStart(ctx, "user-service", "GET", "/internal/users/"+id)
//	result, err := userUsecase.DoSomething(remoteCtx, ...)
//	end(err)
func serviceStart(ctx context.Context, svcName, method, path string) (context.Context, func(error)) {
	ctx, end := telm.Start(ctx, fmt.Sprintf("%s %s", method, path), telm.Client())

	telm.Attr(ctx, telm.F{
		"http.request.method": method,
		"http.url":            "http://" + svcName + path,
		"peer.service":        svcName,
		"net.peer.name":       svcName,
	})

	headers := make(map[string]string)
	telm.InjectHeaders(ctx, headers)

	remoteCtx := telm.ExtractHeaders(context.Background(), headers)

	return remoteCtx, end
}
