# telm SDK

Go observability SDK. Provides a minimal, ergonomic API for the three telemetry signals — **traces**, **logs** and **metrics** — on top of OpenTelemetry, without leaking any OTel types into your business code.

**Transport:** OTLP/HTTP
**Propagation:** W3C TraceContext + Baggage

---

## Table of contents

- [Initialization](#initialization)
- [Types](#types)
- [HTTP Middleware](#http-middleware)
  - [Emitted metrics](#emitted-metrics)
  - [net/http](#nethttp)
  - [chi](#chi)
  - [gorilla/mux](#gorillamux)
  - [Available options](#available-options)
  - [InjectHTTPRequest — outgoing propagation](#injecthttprequest--outgoing-propagation)
- [Tracing](#tracing)
  - [Start — manual spans](#start--manual-spans)
  - [Attr — span attributes](#attr--span-attributes)
  - [Event — span events](#event--span-events)
  - [Span kind](#span-kind)
- [Logging](#logging)
- [Metrics](#metrics)
  - [Count — counter](#count--counter)
  - [Record — histogram](#record--histogram)
  - [Gauge — up/down counter](#gauge--updown-counter)
- [Cross-service propagation](#cross-service-propagation)
- [Complete patterns](#complete-patterns)

---

## Initialization

Call `telm.Init` once at application startup. It returns a shutdown function that must be called on exit to flush any buffered data.

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/locksmithhq/telm-go"
)

func main() {
    ctx := context.Background()

    shutdown, err := telm.Init(ctx,
        telm.WithServiceName("my-service"),
        telm.WithEndpoint("otelcollector:4318"),
    )
    if err != nil {
        slog.Error("telm init failed", "error", err)
        os.Exit(1)
    }
    defer shutdown(context.Background())

    // ... rest of the application
}
```

### Initialization options

| Option | Description | Default |
|---|---|---|
| `WithServiceName(name string)` | Service name shown in traces/logs/metrics | `"unknown"` |
| `WithEndpoint(host:port string)` | OTLP/HTTP collector address (no scheme) | `"localhost:4318"` |

---

## Types

### `telm.F`

`F` is a `map[string]any` used to pass structured fields to logs, span attributes and metric dimensions. Supports `string`, `int`, `int64`, `float64` and `bool`; any other type is converted via `fmt.Sprintf`.

```go
type F map[string]any

// usage examples
telm.F{"user_id": "u_001", "amount": 49.90, "retry": true}
```

### `telm.Level`

Severity constants for use with `telm.Log`:

```go
telm.DEBUG
telm.INFO
telm.WARN
telm.ERROR
telm.FATAL
```

---

## HTTP Middleware

`HTTPMiddleware` automatically instruments any HTTP handler with traces, metrics and logs. Compatible with any router that accepts `func(http.Handler) http.Handler` — no OTel imports required in your business code.

### Emitted metrics

| Metric | Type | Dimensions |
|---|---|---|
| `http.server.requests.total` | counter | method, route, status_code, status_class |
| `http.server.request.duration_ms` | histogram | method, route, status_code, status_class |
| `http.server.request.body_bytes` | histogram | method, route _(only when Content-Length > 0)_ |
| `http.server.response.body_bytes` | histogram | method, route, status_code, status_class |
| `http.server.requests.active` | gauge | method, route |
| `http.server.errors.total` | counter | method, route, status_code, status_class _(4xx/5xx)_ |

**Span attributes** (HTTP semconv):

| Attribute | Example |
|---|---|
| `http.request.method` | `GET` |
| `http.route` | `/api/users/:id` |
| `http.response.status_code` | `200` |
| `http.request.body.size` | `1024` |
| `http.response.body.size` | `512` |
| `net.peer.addr` | `192.168.1.1` (respects X-Forwarded-For) |
| `user_agent.original` | `Mozilla/5.0 …` |
| `server.address` | `api.example.com` |

**Log behavior:**

| Situation | Level |
|---|---|
| 2xx / 3xx within threshold | `INFO` |
| 4xx | `WARN` |
| Duration above `slowThreshold` | `WARN` with field `"slow": true` |
| 5xx | `ERROR` with the error object |

---

### net/http

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /api/users/{id}", usersHandler)
mux.HandleFunc("POST /api/orders", ordersHandler)

// the route pattern (e.g. "GET /api/users/{id}") is extracted automatically
// via r.Pattern available in net/http Go 1.22+
handler := telm.HTTPMiddleware(
    telm.WithSlowThreshold(500 * time.Millisecond),
    telm.WithSkipPaths("/health", "/ready", "/metrics"),
)(mux)

http.ListenAndServe(":8080", handler)
```

---

### chi

```go
// helper to extract the route pattern from chi
func chiRoute(r *http.Request) string {
    if rc := chi.RouteContext(r.Context()); rc != nil {
        if p := rc.RoutePattern(); p != "" {
            return p
        }
    }
    return r.URL.Path
}

r := chi.NewRouter()
r.Use(telm.HTTPMiddleware(
    telm.WithRouteResolver(chiRoute),
    telm.WithSlowThreshold(500 * time.Millisecond),
    telm.WithSkipPaths("/health", "/ready"),
))

r.Get("/api/users/{id}", usersHandler)
r.Post("/api/orders", ordersHandler)

http.ListenAndServe(":8080", r)
```

---

### gorilla/mux

```go
func gorillaMuxRoute(r *http.Request) string {
    if route := mux.CurrentRoute(r); route != nil {
        if tpl, err := route.GetPathTemplate(); err == nil {
            return tpl
        }
    }
    return r.URL.Path
}

router := mux.NewRouter()

// middleware must be applied after routes are registered
// so that CurrentRoute works correctly
router.HandleFunc("/api/users/{id}", usersHandler).Methods("GET")
router.HandleFunc("/api/orders", ordersHandler).Methods("POST")

router.Use(telm.HTTPMiddleware(
    telm.WithRouteResolver(gorillaMuxRoute),
    telm.WithSkipPaths("/health"),
))

http.ListenAndServe(":8080", router)
```

---

### Available options

```go
telm.HTTPMiddleware(
    // custom route pattern extractor (default: r.Pattern or r.URL.Path)
    telm.WithRouteResolver(func(r *http.Request) string {
        return r.URL.Path
    }),

    // duration above which the log becomes WARN (default: 1s)
    telm.WithSlowThreshold(500 * time.Millisecond),

    // paths that bypass all instrumentation
    telm.WithSkipPaths("/health", "/ready", "/metrics", "/favicon.ico"),
)
```

---

### InjectHTTPRequest — outgoing propagation

The middleware automatically extracts the trace context from incoming headers and makes it available via `r.Context()`. Nothing extra is needed in the handler to receive a trace.

To **propagate the trace to other services** in outgoing HTTP calls, use `InjectHTTPRequest`:

```go
func callOtherService(ctx context.Context, url string, body io.Reader) error {
    ctx, end := telm.Start(ctx, "POST "+url, telm.Client())
    defer func() { end(err) }()

    req, err := http.NewRequestWithContext(ctx, "POST", url, body)
    if err != nil {
        return err
    }

    // injects traceparent, tracestate and baggage into the HTTP headers
    telm.InjectHTTPRequest(ctx, req)

    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

> The context passed to `InjectHTTPRequest` already contains the active span — the receiving service will get the propagated trace and its spans will appear as children in the distributed trace.

---

## Tracing

### Start — manual spans

`Start` opens a new span and returns the enriched context and an `end(err)` callback. Passing `nil` marks the span as successful; passing an `error` records the error and marks it as failed.

```go
func Start(ctx context.Context, name string, opts ...SpanStartOption) (context.Context, func(error))
```

**Simple usage:**

```go
ctx, end := telm.Start(ctx, "process.order")
defer end(nil)
```

**With error handling:**

```go
func processOrder(ctx context.Context, id string) (err error) {
    ctx, end := telm.Start(ctx, "process.order")
    defer func() { end(err) }() // propagates the returned error to the span

    // ... logic
    return nil
}
```

**With span kind:**

```go
// incoming HTTP request
ctx, end := telm.Start(ctx, "POST /api/orders", telm.Server())
defer func() { end(err) }()

// database or external service call
ctx, end := telm.Start(ctx, "db.users.findByID", telm.Client())
defer func() { end(err) }()

// internal operation with no I/O
ctx, end := telm.Start(ctx, "usecase.validate.user", telm.Internal())
defer func() { end(err) }()
```

> The context returned by `Start` **must** be passed down to all functions called within the span. It is what guarantees the parent → child span chaining.

---

### Attr — span attributes

`Attr` adds fields to the active span in the context. Accepts one or more `F` maps; multiple `F` values are merged.

```go
func Attr(ctx context.Context, fields ...F)
```

```go
// single F
telm.Attr(ctx, telm.F{"user.id": userID, "route": "/api/orders"})

// multiple F values are merged
telm.Attr(ctx,
    telm.F{"http.request.method": "POST"},
    telm.F{"http.route": "/api/orders"},
    telm.F{"http.response.status_code": 200},
)
```

**Recommended key conventions:**

| Key | Description |
|---|---|
| `"http.request.method"` | HTTP method (`GET`, `POST`, …) |
| `"http.route"` | Request route (`/api/users/:id`) |
| `"http.response.status_code"` | HTTP status code |
| `"http.url"` | Full URL |
| `"db.system"` | Database system (`postgresql`, `mysql`, `redis`) |
| `"db.query.text"` | SQL query text |
| `"net.peer.name"` | Host of the called service |
| `"peer.service"` | Name of the called service |

---

### Event — span events

`Event` records a discrete point in time within the active span. Useful for milestones such as "cache hit", "retry started", "queue published".

```go
func Event(ctx context.Context, name string, fields ...F)
```

```go
// without extra fields
telm.Event(ctx, "token.verified")
telm.Event(ctx, "cache.hit")

// with fields
telm.Event(ctx, "row.fetched", telm.F{"rows": 1})
telm.Event(ctx, "order.inserted", telm.F{"order.id": orderID})
telm.Event(ctx, "charge.succeeded", telm.F{"charge.id": txID, "amount": 99.90})
```

---

### Span kind

Helper functions that return a `SpanStartOption` without exposing OTel types to the caller:

| Function | When to use |
|---|---|
| `telm.Server()` | Request entry point (HTTP server, gRPC server) |
| `telm.Client()` | Outgoing call to database, queue or external service |
| `telm.Internal()` | Internal logic with no I/O (usecase, validation) |

```go
ctx, end := telm.Start(ctx, "GET /api/users", telm.Server())
ctx, end := telm.Start(ctx, "db.users.find", telm.Client())
ctx, end := telm.Start(ctx, "auth.validate", telm.Internal())
```

---

## Logging

All logs are structured and emitted via OTLP, automatically correlated to the active span in the context.

```go
telm.Debug(ctx, "message", fields ...F)
telm.Info(ctx,  "message", fields ...F)
telm.Warn(ctx,  "message", fields ...F)
telm.Error(ctx, "message", err error, fields ...F)
telm.Fatal(ctx, "message", err error, fields ...F)

// custom level
telm.Log(ctx, telm.WARN, "message", fields ...F)
```

```go
// info with fields
telm.Info(ctx, "order created", telm.F{
    "order_id":   orderID,
    "user_id":    userID,
    "amount":     amount,
    "product_id": productID,
})

// warn without a structured error
telm.Warn(ctx, "low stock", telm.F{"product_id": productID, "qty_remaining": 3})

// error with the error object — adds the "error" field automatically
if err != nil {
    telm.Error(ctx, "failed to save transaction", err, telm.F{"tx_id": txID})
}

// debug for internal checkpoints
telm.Debug(ctx, "card retrieved", telm.F{"brand": card.Brand, "last4": card.Last4})

// fatal before process exit
telm.Fatal(ctx, "invalid configuration", err)
```

> `Error` and `Fatal` accept `err error` as their second argument. If non-nil, the `"error"` field is added automatically. To log fields without an error, pass `nil`:
>
> ```go
> telm.Error(ctx, "rate limit reached", nil, telm.F{"route": route})
> ```

---

## Metrics

### Count — counter

`Count` increments an `Int64Counter` by n. Use to count events that only grow: requests, errors, emails sent.

```go
func Count(ctx context.Context, name string, n int64, fields ...F)
```

```go
// simple increment
telm.Count(ctx, "http.requests.total", 1)

// with dimensions (labels)
telm.Count(ctx, "http.requests.total", 1, telm.F{
    "method": "POST",
    "route":  "/api/orders",
    "status": 200,
})

// counting failures
telm.Count(ctx, "payments.failed", 1, telm.F{"reason": "card_declined"})

// increment by more than 1
telm.Count(ctx, "bytes.processed", int64(len(payload)))
```

---

### Record — histogram

`Record` records a `float64` value in a `Float64Histogram`. Use for durations, sizes, monetary values — anything that needs a distribution (p50, p95, p99).

```go
func Record(ctx context.Context, name string, value float64, fields ...F)
```

```go
// duration in milliseconds
start := time.Now()
// ... operation
telm.Record(ctx, "db.query.duration_ms", float64(time.Since(start).Milliseconds()),
    telm.F{"table": "orders"},
)

// transaction value
telm.Record(ctx, "payments.amount", amount, telm.F{"currency": "USD"})

// payload size
telm.Record(ctx, "http.request.size_bytes", float64(r.ContentLength))
```

---

### Gauge — up/down counter

`Gauge` adjusts an `Int64UpDownCounter`. Use for values that go up and down: active connections, workers in use, items in queue.

```go
func Gauge(ctx context.Context, name string, delta int64, fields ...F)
```

```go
// increment on enter
telm.Gauge(ctx, "requests.active", +1, telm.F{"worker": workerID})
// ... process
// decrement on exit
telm.Gauge(ctx, "requests.active", -1, telm.F{"worker": workerID})

// database connections
telm.Gauge(ctx, "db.connections.open", +1)
defer telm.Gauge(ctx, "db.connections.open", -1)

// queue depth
telm.Gauge(ctx, "queue.depth", int64(len(items)), telm.F{"queue": "orders"})
```

---

## Cross-service propagation

To propagate the trace context in real HTTP calls, use `InjectHeaders` when sending and `ExtractHeaders` when receiving.

```go
func InjectHeaders(ctx context.Context, headers map[string]string)
func ExtractHeaders(ctx context.Context, headers map[string]string) context.Context
```

**Client (caller):**

```go
func callService(ctx context.Context, url string) error {
    ctx, end := telm.Start(ctx, "POST "+url, telm.Client())
    defer func() { end(err) }()

    req, _ := http.NewRequestWithContext(ctx, "POST", url, body)

    // inject traceparent, tracestate and baggage into the HTTP headers
    headers := make(map[string]string)
    telm.InjectHeaders(ctx, headers)
    for k, v := range headers {
        req.Header.Set(k, v)
    }

    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

**Server (receiver):**

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // extract the trace context from incoming headers
    headers := make(map[string]string)
    for k := range r.Header {
        headers[k] = r.Header.Get(k)
    }
    ctx := telm.ExtractHeaders(r.Context(), headers)

    ctx, end := telm.Start(ctx, r.Method+" "+r.URL.Path, telm.Server())
    defer end(nil)

    // from here, all child spans are chained to the caller's trace
}
```

---

## Complete patterns

### HTTP Handler

```go
func (h *handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    // extract the trace propagated by the caller (API gateway, browser, etc.)
    headers := make(map[string]string)
    for k := range r.Header {
        headers[k] = r.Header.Get(k)
    }
    ctx := telm.ExtractHeaders(r.Context(), headers)

    ctx, end := telm.Start(ctx, r.Method+" "+r.URL.Path, telm.Server())
    defer end(nil)

    telm.Attr(ctx, telm.F{
        "http.request.method": r.Method,
        "http.route":          "/api/orders",
    })

    start := time.Now()

    order, err := h.usecase.CreateOrder(ctx, r.Body)
    statusCode := 201
    if err != nil {
        statusCode = 500
        telm.Error(ctx, "create order failed", err)
    }

    telm.Attr(ctx, telm.F{"http.response.status_code": statusCode})
    telm.Count(ctx, "http.requests.total", 1, telm.F{
        "method": r.Method,
        "route":  "/api/orders",
        "status": statusCode,
    })
    telm.Record(ctx, "http.request.duration_ms",
        float64(time.Since(start).Milliseconds()),
        telm.F{"route": "/api/orders"},
    )

    w.WriteHeader(statusCode)
}
```

---

### Database query

```go
func (r *orderRepo) Insert(ctx context.Context, o *Order) (id string, err error) {
    ctx, end := telm.Start(ctx, "db.orders.insert", telm.Client())
    defer func() { end(err) }()

    telm.Attr(ctx, telm.F{
        "db.system":     "postgresql",
        "db.query.text": "INSERT INTO orders (user_id, amount) VALUES ($1, $2) RETURNING id",
        "db.user.id":    o.UserID,
        "db.amount":     o.Amount,
    })

    start := time.Now()
    err = r.db.QueryRowContext(ctx,
        "INSERT INTO orders (user_id, amount) VALUES ($1, $2) RETURNING id",
        o.UserID, o.Amount,
    ).Scan(&id)

    telm.Record(ctx, "db.query.duration_ms",
        float64(time.Since(start).Milliseconds()),
        telm.F{"table": "orders", "op": "insert"},
    )

    if err != nil {
        telm.Error(ctx, "db insert failed", err, telm.F{"table": "orders"})
        return "", err
    }

    telm.Event(ctx, "order.inserted", telm.F{"order.id": id})
    return id, nil
}
```

---

### External service call (HTTP client)

```go
func (g *stripeGateway) Charge(ctx context.Context, token string, amount float64) (txID string, err error) {
    ctx, end := telm.Start(ctx, "http.stripe.charge", telm.Client())
    defer func() { end(err) }()

    telm.Attr(ctx, telm.F{
        "http.request.method": "POST",
        "http.url":            "https://api.stripe.com/v1/charges",
        "peer.service":        "stripe",
        "net.peer.name":       "api.stripe.com",
        "stripe.amount":       amount,
    })

    // inject the trace context into the real HTTP request
    req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.stripe.com/v1/charges", body)
    headers := make(map[string]string)
    telm.InjectHeaders(ctx, headers)
    for k, v := range headers {
        req.Header.Set(k, v)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        telm.Count(ctx, "stripe.charges.failed", 1, telm.F{"reason": "network"})
        return "", err
    }

    txID = resp.Header.Get("X-Transaction-Id")
    telm.Event(ctx, "stripe.charge.succeeded", telm.F{"charge.id": txID})
    telm.Count(ctx, "stripe.charges.total", 1, telm.F{"status": "success"})
    telm.Record(ctx, "stripe.charge.amount", amount)
    return txID, nil
}
```

---

### Worker with concurrency gauge

```go
func runWorker(ctx context.Context, id int, jobs <-chan Job) {
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-jobs:
            if !ok {
                return
            }

            telm.Gauge(ctx, "workers.busy", +1, telm.F{"worker_id": id})

            ctx, end := telm.Start(ctx, "worker.process", telm.Internal())
            telm.Attr(ctx, telm.F{"job.id": job.ID, "job.type": job.Type})

            err := processJob(ctx, job)
            end(err)

            if err != nil {
                telm.Count(ctx, "jobs.failed", 1, telm.F{"type": job.Type})
            } else {
                telm.Count(ctx, "jobs.completed", 1, telm.F{"type": job.Type})
            }

            telm.Gauge(ctx, "workers.busy", -1, telm.F{"worker_id": id})
        }
    }
}
```
