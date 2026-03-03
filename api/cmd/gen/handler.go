package main

// handler.go — entry layer (simulated HTTP handler).
// Represents the api-gateway that receives requests and orchestrates services.

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/locksmithhq/telm-go"
)

// apiHandler represents the api-gateway — the request entry point.
type apiHandler struct {
	userUC         *userUsecase
	orderUC        *orderUsecase
	notificationUC *notificationUsecase
}

// newAPIHandler wires the dependency graph (manual DI).
func newAPIHandler(gatewaySvcName string) *apiHandler {
	return &apiHandler{
		userUC: &userUsecase{
			repo: &userRepository{serviceName: gatewaySvcName},
		},
		orderUC: &orderUsecase{
			repo: &orderRepository{serviceName: gatewaySvcName},
			payment: &paymentUsecase{
				repo:   &paymentRepository{serviceName: gatewaySvcName},
				stripe: &stripeGateway{},
			},
		},
		notificationUC: &notificationUsecase{
			email: &emailGateway{},
		},
	}
}

// HandlePlaceOrder processes the full order creation request.
// Span tree:
//
//	POST /api/orders                                    (SERVER — api-gateway)
//	  auth.validate                                     (INTERNAL)
//	  GET /internal/users/:id                           (CLIENT → user-service)
//	    usecase.user.validate                           (INTERNAL — user-service)
//	      db.users.findByID                             (CLIENT)
//	      db.permissions.check                         (CLIENT)
//	  usecase.order.create                              (INTERNAL — api-gateway)
//	    db.inventory.check                              (CLIENT)
//	    db.orders.insert                                (CLIENT)
//	    POST /internal/payments/charge                  (CLIENT → payment-service)
//	      usecase.payment.charge                        (SERVER — payment-service)
//	        db.cards.getDefault                         (CLIENT)
//	        http.stripe.charge                          (CLIENT)
//	        db.transactions.save                        (CLIENT)
//	  POST /internal/notifications/send                 (CLIENT → notification-service)
//	    usecase.notification.send                       (SERVER — notification-service)
//	      http.email.send                               (CLIENT)
func (h *apiHandler) HandlePlaceOrder(ctx context.Context, method, route string, statusCode int) (err error) {
	ctx, end := telm.Start(ctx, method+" "+route, telm.Server())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"http.request.method": method,
		"http.route":          route,
		"layer":               "handler",
		"service":             "api-gateway",
	})

	if authErr := h.validateAuth(ctx); authErr != nil {
		telm.Warn(ctx, "auth failed", telm.F{"route": route})
		return authErr
	}

	userID := randomUserID()
	remoteCtx, callEnd := serviceStart(ctx, "user-service", "GET", "/internal/users/"+userID)
	var usr *user
	usr, err = h.userUC.ValidateUser(remoteCtx, userID)
	callEnd(err)
	if err != nil {
		statusCode = 401
		telm.Warn(ctx, "user validation failed", telm.F{"user_id": userID, "status": statusCode})
		return nil
	}

	productID := randomProductID()
	amount := 50.0 + float64(rand.IntN(950))

	orderID, orderErr := h.orderUC.CreateOrder(ctx, usr, productID, amount)
	if orderErr != nil {
		statusCode = 500
		telm.Error(ctx, "order creation failed", orderErr, telm.F{"product_id": productID, "amount": amount})
		telm.Attr(ctx, telm.F{"http.response.status_code": statusCode})
		return orderErr
	}

	_ = h.notificationUC.Notify(ctx, usr.Email, fmt.Sprintf("Order %s confirmed!", orderID), "order_confirmation")

	telm.Attr(ctx, telm.F{"http.response.status_code": statusCode})
	telm.Count(ctx, "http.requests.total", 1, telm.F{"method": method, "route": route, "status": statusCode})
	telm.Info(ctx, "request handled", telm.F{
		"method":     method,
		"route":      route,
		"status":     statusCode,
		"user_id":    userID,
		"order_id":   orderID,
		"product_id": productID,
		"amount":     amount,
	})
	return nil
}

// HandleSimple handles routes that don't create orders (GET resources).
func (h *apiHandler) HandleSimple(ctx context.Context, method, route string, statusCode int) error {
	ctx, end := telm.Start(ctx, method+" "+route, telm.Server())
	defer end(nil)

	telm.Attr(ctx, telm.F{
		"http.request.method":       method,
		"http.route":                route,
		"http.response.status_code": statusCode,
		"layer":                     "handler",
	})

	if err := h.validateAuth(ctx); err != nil {
		return nil
	}

	userID := randomUserID()
	_, err := h.userUC.ValidateUser(ctx, userID)

	if err != nil || statusCode >= 400 {
		telm.Warn(ctx, "simple request error", telm.F{"method": method, "route": route, "status": statusCode})
	} else {
		telm.Info(ctx, "request handled", telm.F{"method": method, "route": route, "status": statusCode})
	}

	telm.Count(ctx, "http.requests.total", 1, telm.F{"method": method, "route": route, "status": statusCode})
	return nil
}

// validateAuth simulates JWT/API key validation.
func (h *apiHandler) validateAuth(ctx context.Context) (err error) {
	ctx, end := telm.Start(ctx, "auth.validate", telm.Internal())
	defer func() { end(err) }()

	telm.Attr(ctx, telm.F{
		"auth.type": "jwt",
		"layer":     "middleware",
	})
	sleep(2, 8)
	if rand.IntN(25) == 0 {
		err = fmt.Errorf("invalid or expired token")
		return
	}
	telm.Event(ctx, "token.verified")
	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────

var (
	userIDs    = []string{"u_001", "u_002", "u_003", "u_004", "u_005"}
	productIDs = []string{"prod_shoes", "prod_shirt", "prod_bag", "prod_watch", "prod_book"}
)

func randomUserID() string    { return userIDs[rand.IntN(len(userIDs))] }
func randomProductID() string { return productIDs[rand.IntN(len(productIDs))] }
