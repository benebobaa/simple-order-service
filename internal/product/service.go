// Package product implements product management.
package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/benebobaa/simple-order-service/internal/apperr"
	"github.com/benebobaa/simple-order-service/internal/store"
	"github.com/benebobaa/simple-order-service/internal/store/sqlc"
)

// Service implements product management.
type Service struct {
	store *store.Store
}

// NewService creates a Service.
func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

// Input carries the fields used to create or update a product. SKU is only
// meaningful on create: it is the immutable business key.
type Input struct {
	SKU         string
	Name        string
	Description string
	Price       int64
	Stock       int32
}

// Create stores a new product. The SKU is normalized to uppercase and must be
// unique across products.
func (s *Service) Create(ctx context.Context, in Input) (sqlc.Product, error) {
	in, err := normalizeCreateInput(in)
	if err != nil {
		return sqlc.Product{}, err
	}

	// Friendly pre-check; the unique constraint below remains the source of truth.
	if _, err := s.store.Queries().GetProductBySKU(ctx, in.SKU); err == nil {
		return sqlc.Product{}, apperr.Conflict(apperr.CodeSKUExists, "a product with this sku already exists", nil)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Product{}, fmt.Errorf("lookup product by sku: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Product{}, fmt.Errorf("generate product id: %w", err)
	}

	created, err := s.store.Queries().CreateProduct(ctx, sqlc.CreateProductParams{
		ID:          id,
		Sku:         in.SKU,
		Name:        in.Name,
		Description: in.Description,
		Price:       in.Price,
		Stock:       in.Stock,
	})
	if err != nil {
		if store.IsUniqueViolation(err) {
			return sqlc.Product{}, apperr.Conflict(apperr.CodeSKUExists, "a product with this sku already exists", nil)
		}
		return sqlc.Product{}, fmt.Errorf("create product: %w", err)
	}
	return created, nil
}

// Get returns a single product by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (sqlc.Product, error) {
	product, err := s.store.Queries().GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Product{}, apperr.NotFound(apperr.CodeProductNotFound, "product not found")
		}
		return sqlc.Product{}, fmt.Errorf("get product: %w", err)
	}
	return product, nil
}

// List returns a page of products plus the total number of products.
func (s *Service) List(ctx context.Context, limit, offset int32) ([]sqlc.Product, int64, error) {
	products, err := s.store.Queries().ListProducts(ctx, sqlc.ListProductsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}

	total, err := s.store.Queries().CountProducts(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}
	return products, total, nil
}

// Update replaces the mutable fields of a product. The SKU cannot be changed.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in Input) (sqlc.Product, error) {
	in, err := normalizeInput(in)
	if err != nil {
		return sqlc.Product{}, err
	}

	product, err := s.store.Queries().UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
		Price:       in.Price,
		Stock:       in.Stock,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Product{}, apperr.NotFound(apperr.CodeProductNotFound, "product not found")
		}
		return sqlc.Product{}, fmt.Errorf("update product: %w", err)
	}
	return product, nil
}

func normalizeCreateInput(in Input) (Input, error) {
	in, err := normalizeInput(in)
	if err != nil {
		return in, err
	}

	in.SKU = normalizeSKU(in.SKU)
	if len(in.SKU) < 2 || len(in.SKU) > 64 {
		return in, apperr.Validation("sku must be 2-64 characters", map[string]any{"sku": "length"})
	}
	return in, nil
}

// normalizeSKU returns the canonical SKU form: trimmed and uppercased.
func normalizeSKU(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func normalizeInput(in Input) (Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return in, apperr.Validation("name must not be blank", map[string]any{"name": "required"})
	}
	if in.Price < 0 {
		return in, apperr.Validation("price must not be negative", map[string]any{"price": "gte=0"})
	}
	if in.Stock < 0 {
		return in, apperr.Validation("stock must not be negative", map[string]any{"stock": "gte=0"})
	}
	return in, nil
}
