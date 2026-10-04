// Package order implements order placement, listing and cancellation with
// atomic stock management.
package order

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/benebobaa/simple-order-service/internal/apperr"
	"github.com/benebobaa/simple-order-service/internal/store"
	"github.com/benebobaa/simple-order-service/internal/store/sqlc"
)

// maxInt64 bounds order arithmetic: any line or total beyond it is rejected as
// invalid input instead of wrapping silently into a negative amount.
const maxInt64 = 1<<63 - 1

// Service implements order placement and stock management.
type Service struct {
	store *store.Store
}

// NewService creates a Service.
func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

// ItemInput is a single requested product/quantity pair, referencing the
// product by its canonical SKU.
type ItemInput struct {
	SKU      string
	Quantity int32
}

// Detail is an order together with its items and product names.
type Detail struct {
	Order sqlc.Order
	Items []sqlc.ListOrderItemsWithProductNameByOrderIDsRow
}

// Create places an order for the given user, reserving stock for every item
// atomically. If any item cannot be fulfilled, the whole order is rejected and
// no stock is reserved.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, items []ItemInput) (*Detail, error) {
	items, err := normalizeItems(items)
	if err != nil {
		return nil, err
	}

	var (
		orderID uuid.UUID
		detail  *Detail
	)
	err = s.store.WithTx(ctx, func(q *sqlc.Queries) error {
		skus := make([]string, 0, len(items))
		for _, item := range items {
			skus = append(skus, item.SKU)
		}

		// Lock every requested product in a deterministic (id-sorted) order so
		// concurrent multi-item orders cannot deadlock, then validate stock.
		locked, err := q.ListProductsBySKUsForUpdate(ctx, skus)
		if err != nil {
			return fmt.Errorf("lock products: %w", err)
		}
		products := make(map[string]sqlc.Product, len(locked))
		for _, product := range locked {
			products[product.Sku] = product
		}

		var totalPrice int64
		subtotals := make(map[string]int64, len(items))
		for i, item := range items {
			product, ok := products[item.SKU]
			if !ok {
				return apperr.NotFound(apperr.CodeProductNotFound, "product not found").
					WithDetails(map[string]any{
						"items": fmt.Sprintf("items[%d].sku", i),
						"sku":   item.SKU,
					})
			}
			if product.Stock < item.Quantity {
				return apperr.Conflict(apperr.CodeInsufficientStock,
					fmt.Sprintf("insufficient stock for product %q", product.Name),
					map[string]any{
						"items":        fmt.Sprintf("items[%d]", i),
						"product_id":   product.ID,
						"sku":          product.Sku,
						"product_name": product.Name,
						"requested":    item.Quantity,
						"available":    product.Stock,
					})
			}

			subtotal, fits := checkedSubtotal(product.Price, item.Quantity)
			if !fits || totalPrice > maxInt64-subtotal {
				return apperr.Validation("order total exceeds the maximum supported value",
					map[string]any{"items": fmt.Sprintf("items[%d]", i)})
			}
			totalPrice += subtotal
			subtotals[item.SKU] = subtotal
		}

		orderID, err = uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate order id: %w", err)
		}
		created, err := q.CreateOrder(ctx, sqlc.CreateOrderParams{
			ID:         orderID,
			UserID:     userID,
			TotalPrice: totalPrice,
		})
		if err != nil {
			return fmt.Errorf("create order: %w", err)
		}

		for _, item := range items {
			product := products[item.SKU]

			itemID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate order item id: %w", err)
			}
			if _, err := q.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
				ID:        itemID,
				OrderID:   orderID,
				ProductID: product.ID,
				Sku:       product.Sku,
				Quantity:  item.Quantity,
				UnitPrice: product.Price,
				Subtotal:  subtotals[item.SKU],
			}); err != nil {
				return fmt.Errorf("create order item: %w", err)
			}

			affected, err := q.DecreaseProductStock(ctx, sqlc.DecreaseProductStockParams{
				ID:       product.ID,
				Quantity: item.Quantity,
			})
			if err != nil {
				return fmt.Errorf("reserve stock: %w", err)
			}
			if affected != 1 {
				return fmt.Errorf("reserve stock: expected 1 row affected, got %d", affected)
			}
		}

		// Build the response inside the transaction so a post-commit read
		// failure can never report an order as failed after it was persisted.
		rows, err := q.ListOrderItemsWithProductNameByOrderIDs(ctx, []uuid.UUID{orderID})
		if err != nil {
			return fmt.Errorf("list order items: %w", err)
		}
		detail = &Detail{Order: created, Items: rows}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return detail, nil
}

// checkedSubtotal multiplies a non-negative price by a positive quantity and
// reports false instead of wrapping when the result would overflow int64.
func checkedSubtotal(price int64, quantity int32) (int64, bool) {
	if price > maxInt64/int64(quantity) {
		return 0, false
	}
	return price * int64(quantity), true
}

// normalizeItems validates and canonicalizes the requested items: at least one
// item, a non-empty SKU, positive quantities and no duplicated SKUs.
func normalizeItems(items []ItemInput) ([]ItemInput, error) {
	if len(items) == 0 {
		return nil, apperr.Validation("an order must contain at least one item", map[string]any{"items": "required"})
	}

	seen := make(map[string]struct{}, len(items))
	for i := range items {
		items[i].SKU = normalizeSKU(items[i].SKU)
		item := items[i]
		key := fmt.Sprintf("items[%d]", i)

		if item.SKU == "" {
			return nil, apperr.Validation("sku is required", map[string]any{key + ".sku": "required"})
		}
		if item.Quantity <= 0 {
			return nil, apperr.Validation("quantity must be greater than zero", map[string]any{key + ".quantity": "gt=0"})
		}
		if _, duplicate := seen[item.SKU]; duplicate {
			return nil, apperr.Validation("duplicate product in order", map[string]any{key + ".sku": "duplicate"})
		}
		seen[item.SKU] = struct{}{}
	}
	return items, nil
}

// normalizeSKU returns the canonical SKU form: trimmed and uppercased.
func normalizeSKU(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}
