package product

import (
	"time"

	"github.com/google/uuid"

	"github.com/benebobaa/simple-order-service/internal/store/sqlc"
)

// createRequest is the payload for creating a product. Numeric fields use
// pointers so "0" can be supplied explicitly while a missing field is still
// rejected as invalid. The SKU is the immutable, human-facing business key.
type createRequest struct {
	SKU         string `json:"sku" binding:"required,min=2,max=64"`
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description" binding:"max=2000"`
	Price       *int64 `json:"price" binding:"required,gte=0"`
	Stock       *int32 `json:"stock" binding:"required,gte=0"`
}

// updateRequest is the payload for replacing a product's mutable fields. The
// SKU is intentionally absent: the business key never changes.
type updateRequest struct {
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description" binding:"max=2000"`
	Price       *int64 `json:"price" binding:"required,gte=0"`
	Stock       *int32 `json:"stock" binding:"required,gte=0"`
}

func (req createRequest) input() Input {
	return Input{
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Price:       *req.Price,
		Stock:       *req.Stock,
	}
}

func (req updateRequest) input() Input {
	return Input{
		Name:        req.Name,
		Description: req.Description,
		Price:       *req.Price,
		Stock:       *req.Stock,
	}
}

// Response is the public representation of a product.
type Response struct {
	ID          uuid.UUID `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int64     `json:"price"`
	Stock       int32     `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newResponse(product sqlc.Product) Response {
	return Response{
		ID:          product.ID,
		SKU:         product.Sku,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}
