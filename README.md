# Simple Order Service

REST API for an order management service in Go: users, products with stock,
and orders that reserve stock atomically on creation and return it exactly
once on cancellation.

## Tech stack

| Concern | Choice |
| --- | --- |
| Language | Go 1.27 (module `github.com/benebobaa/simple-order-service`) |
| HTTP | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL 16 + pgx pool |
| Queries | [sqlc](https://sqlc.dev), generated from `db/queries` |
| Migrations | [goose](https://github.com/pressly/goose), applied on startup by default |
| Auth | JWT (HS256) + bcrypt passwords |
| Validation | Gin binding + service checks + database constraints |
| Logging | `log/slog`, structured JSON |
| Tests / lint | testify suites + testcontainers-go (real Postgres, real router); golangci-lint |

`sqlc`, `goose` and the Go toolchain are pinned through `go.mod`, so
`make sqlc` and `make migrate-up` work without global installs.

## Getting started

Requires Go 1.24+ (the module pins 1.27), Docker and GNU Make.

```bash
cp .env.example .env   # defaults match docker-compose
make compose-up        # PostgreSQL 16 on :5432
make run               # applies migrations, serves :8080
curl localhost:8080/healthz
```

| Variable | Default | Notes |
| --- | --- | --- |
| `DATABASE_URL` | — (required) | PostgreSQL DSN |
| `JWT_SECRET` | — (required) | signing key |
| `PORT`, `APP_ENV` | `8080`, `development` | |
| `JWT_TTL`, `LOG_LEVEL` | `24h`, `info` | |
| `AUTO_MIGRATE` | `true` | run goose migrations at startup |
| `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME` | pgx defaults | pool tuning; blank = default |

## API reference

Base path `/v1`. Authenticated routes expect `Authorization: Bearer <token>`.
IDs are UUID v7, SKUs are uppercase business keys, prices are integer IDR.

| Method | Path | Auth | Description |
| --- | --- | :--: | --- |
| POST | `/v1/auth/register` | | Create account → user + token |
| POST | `/v1/auth/login` | | Sign in → user + token |
| GET | `/v1/auth/me` | ✔ | Current user profile |
| POST | `/v1/products` | ✔ | Create product (unique, immutable `sku`) |
| GET | `/v1/products` | | List products (`limit`, `offset`) |
| GET | `/v1/products/:id` | | Get product |
| PUT | `/v1/products/:id` | ✔ | Update product (full replace) |
| POST | `/v1/orders` | ✔ | Create order (≤ 100 items, referenced by SKU); reserves stock |
| GET | `/v1/orders` | ✔ | Caller's orders (`status`, `limit`, `offset`) |
| GET | `/v1/orders/:id` | ✔ | Order detail with items |
| POST | `/v1/orders/:id/cancel` | ✔ | Cancel; returns stock (`409` if already cancelled) |
| GET | `/healthz`, `/readyz` | | Liveness, readiness (readiness pings the database) |

Successes are `{"data": …}` (lists add `meta`), failures `{"error": …}`; the
HTTP status is authoritative — there is no duplicated `status` field. Every
response carries an `X-Request-ID` header, echoed in error bodies for tracing.

```jsonc
{ "data": { "id": "…", "sku": "KOPI-ARABIKA-1KG" } }
{ "data": [ … ], "meta": { "total": 3, "limit": 20, "offset": 0 } }
{ "error": { "code": "INSUFFICIENT_STOCK", "message": "…",
             "details": { "sku": "…", "requested": 5, "available": 2 },
             "request_id": "0199a0f3-…" } }
```

| Status | When |
| --- | --- |
| 400 | validation (`VALIDATION_ERROR`, per-field details) |
| 401 | missing/invalid token, bad credentials (`INVALID_CREDENTIALS`) |
| 404 | unknown resource; other users' orders hidden as 404 |
| 409 | `INSUFFICIENT_STOCK`, `ORDER_ALREADY_CANCELLED`, `EMAIL_ALREADY_EXISTS`, `SKU_ALREADY_EXISTS` |
| 500 | unexpected (logged, no internals leaked) |

List defaults: `limit=20` (max 100), `offset=0`; unparsable values fall back,
oversized values are clamped.

## Project layout

```
cmd/api/          thin entrypoint (main only)
internal/
  app/            composition root: wiring + lifecycle
  auth/ product/ order/    feature modules (dto, handler, service, routes)
  httpapi/        router mounting the modules
  web/            shared HTTP helpers (envelope, binding, pagination)
  apperr/         typed errors + API codes
  config/         env configuration (fail fast)
  store/          pgx pool, transaction helper, sqlc/ generated queries
db/               goose migrations + sqlc queries (single schema source)
test/integration/ full-stack tests (real router + real Postgres)
docs/             Postman collection
```

Dependency direction: feature modules → `store`/`web`, `httpapi`
composes features, and `app` wires everything.

## Database & migrations

Tables: `users`, `products` (unique uppercase `sku`, `stock >= 0`), `orders`
(`pending`/`cancelled`), `order_items` (`UNIQUE (order_id, product_id)`).

```bash
make migrate-up       # apply pending migrations
make migrate-down     # roll back one
make migrate-status   # show versions
make migrate-reset    # roll back everything
make sqlc             # regenerate internal/store/sqlc after editing db/queries
```

Migrations double as the sqlc schema source — no duplicated schema file.

## Testing

```bash
make test-unit         # unit tests, no Docker
make test-integration  # testcontainers: real PostgreSQL, real HTTP router
make test              # both
make coverage          # integration coverage: prints total, opens HTML report
make check             # one-shot gate: gofmt + vet + lint + all tests
```

Coverage spans auth, products, orders and the stock rules including races;
`make coverage` mirrors CI and currently measures ~72% (integration-only,
`-coverpkg=./...`).

### Sanity checks (k6)

`test/sanity/` is a post-deploy suite that walks a deployed environment end
to end over HTTP. Files are split per resource — `health.js`, `auth.js`,
`products.js`, `orders.js` — and cover the happy path plus the main error
contracts (`400`, `401`, `404`, `409`). It needs no credentials: every run
registers its own users and creates its own products.

```bash
make sanity                                    # whole suite against staging
make sanity SANITY_URL=http://localhost:8080   # any environment
k6 run test/sanity/orders.js                   # a single area, standalone
```

## Postman collection

Import [`docs/simple-order-service.postman_collection.json`](docs/simple-order-service.postman_collection.json)

## CI

`.github/workflows/ci.yml` runs four jobs on push and pull requests:

- **Lint** — `gofmt` check + `golangci-lint`
- **Unit** — `go test ./internal/...`
- **Integration + coverage** — testcontainers suite; fails below the
  repository variable `COVERAGE_MINIMUM` (default `70`), profile uploaded
- **Vulnerability scan** — `govulncheck` + Trivy (HIGH/CRITICAL); report-only for now

## Deploy

Pushing a `v*` tag runs `.github/workflows/deploy.yml`: the `Dockerfile` image
is built for `linux/amd64`, pushed to `ghcr.io/benebobaa/simple-order-service`
and deployed with `kubeletto deploy … --wait`. The k6 sanity suite then runs
against the live URL; a failed check fails the workflow and flags the release
for rollback with `kubeletto rollback simple-order-service`.
