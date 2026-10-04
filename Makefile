GO          ?= go
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/orders?sslmode=disable
SANITY_URL  ?= https://simple-order-service.kubeletto.app

.PHONY: run build fmt fmt-check vet sqlc migrate-up migrate-down migrate-status migrate-reset \
        test test-unit test-integration coverage sanity lint check compose-up compose-down

## Development

run:
	$(GO) run ./cmd/api

build:
	$(GO) build -o bin/api ./cmd/api

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

## Code generation

sqlc:
	$(GO) tool sqlc generate

## Database migrations (goose)

migrate-up:
	$(GO) tool goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	$(GO) tool goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-status:
	$(GO) tool goose -dir db/migrations postgres "$(DATABASE_URL)" status

migrate-reset:
	$(GO) tool goose -dir db/migrations postgres "$(DATABASE_URL)" reset

## Testing

test:
	$(GO) test ./... -count=1

test-unit:
	$(GO) test ./internal/... -count=1

test-integration:
	$(GO) test ./test/integration/... -count=1 -v

# coverage runs the same measurement as CI (integration tests across the whole
# module), prints the total and opens the annotated HTML report.
coverage:
	$(GO) test ./test/integration/ -count=1 -coverpkg=./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out | tail -1
	$(GO) tool cover -html=coverage.out -o coverage.html
	open coverage.html || echo "open coverage.html in your browser"

# sanity runs the post-deploy k6 suite against a deployed environment.
sanity:
	k6 run --env BASE_URL="$(SANITY_URL)" test/sanity/sanity.js

## Quality

lint:
	golangci-lint run

fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		echo "gofmt needed in:"; echo "$$files"; exit 1; \
	fi

# check is the full quality gate: formatting, vet, lint and all tests.
check: fmt-check vet lint test

## Infrastructure

compose-up:
	docker compose up -d

compose-down:
	docker compose down
