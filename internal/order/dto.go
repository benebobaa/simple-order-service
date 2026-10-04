package order

import (
	"time"

	"github.com/google/uuid"
)

type createOrderRequest struct {
	Items []createOrderItemRequest `json:"items" binding:"required,min=1,max=100,dive"`
}

// createOrderItemRequest references the product by its canonical SKU; the
// service resolves it to the internal product ID under lock.
type createOrderItemRequest struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int32  `json:"quantity" binding:"required,gt=0"`
}

// ItemResponse is the public representation of an order line. It exposes the
// immutable product ID alongside the SKU snapshot taken when the order was
// placed, so historical orders remain readable.
type ItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	SKU         string    `json:"sku"`
	ProductName string    `json:"product_name"`
	Quantity    int32     `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	Subtotal    int64     `json:"subtotal"`
}

// Response is the public representation of an order.
type Response struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Status      string         `json:"status"`
	TotalPrice  int64          `json:"total_price"`
	Items       []ItemResponse `json:"items"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CancelledAt *time.Time     `json:"cancelled_at,omitempty"`
}

func newResponse(detail *Detail) Response {
	items := make([]ItemResponse, 0, len(detail.Items))
	for _, item := range detail.Items {
		items = append(items, ItemResponse{
			ID:          item.ID,
			ProductID:   item.ProductID,
			SKU:         item.Sku,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.Subtotal,
		})
	}

	return Response{
		ID:          detail.Order.ID,
		UserID:      detail.Order.UserID,
		Status:      detail.Order.Status,
		TotalPrice:  detail.Order.TotalPrice,
		Items:       items,
		CreatedAt:   detail.Order.CreatedAt,
		UpdatedAt:   detail.Order.UpdatedAt,
		CancelledAt: detail.Order.CancelledAt,
	}
}
