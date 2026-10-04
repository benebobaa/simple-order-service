// Package order implements order placement, listing and cancellation with
// atomic stock management.
package order

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

// Get returns one order with its items, scoped to the owning user.
func (s *Service) Get(ctx context.Context, userID, orderID uuid.UUID) (*Detail, error) {
	return s.getDetail(ctx, userID, orderID)
}

// List returns a page of the user's orders plus the total count. The optional
// status filter accepts "pending" or "cancelled".
func (s *Service) List(ctx context.Context, userID uuid.UUID, status *string, limit, offset int32) ([]sqlc.Order, int64, error) {
	normalizedStatus, err := normalizeStatus(status)
	if err != nil {
		return nil, 0, err
	}

	orders, err := s.store.Queries().ListOrdersByUser(ctx, sqlc.ListOrdersByUserParams{
		UserID:    userID,
		Status:    normalizedStatus,
		RowLimit:  limit,
		RowOffset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	total, err := s.store.Queries().CountOrdersByUser(ctx, sqlc.CountOrdersByUserParams{
		UserID: userID,
		Status: normalizedStatus,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}
	return orders, total, nil
}

func normalizeStatus(status *string) (*string, error) {
	if status == nil || strings.TrimSpace(*status) == "" {
		return nil, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(*status))
	if normalized != "pending" && normalized != "cancelled" {
		return nil, apperr.Validation("invalid status filter", map[string]any{"status": "in=pending cancelled"})
	}
	return &normalized, nil
}

func (s *Service) getDetail(ctx context.Context, userID, orderID uuid.UUID) (*Detail, error) {
	order, err := s.store.Queries().GetOrderByIDForUser(ctx, sqlc.GetOrderByIDForUserParams{
		ID:     orderID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound(apperr.CodeOrderNotFound, "order not found")
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	items, err := s.store.Queries().ListOrderItemsWithProductNameByOrderIDs(ctx, []uuid.UUID{orderID})
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}

	return &Detail{Order: order, Items: items}, nil
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
