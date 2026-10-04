package integration

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	orderapi "github.com/benebobaa/simple-order-service/internal/order"
)

type orderSuite struct {
	baseSuite
}

func TestOrderSuite(t *testing.T) {
	suite.Run(t, new(orderSuite))
}

func (s *orderSuite) TestCreateOrder_ReservesStockAndSnapshotsPrices() {
	token := registerUser(s.T()).Token
	coffee := createProduct(s.T(), token, map[string]any{"sku": "COFFEE-1KG", "name": "Coffee", "price": 15000, "stock": 10})
	tea := createProduct(s.T(), token, map[string]any{"sku": "tea-250g", "name": "Tea", "price": 8000, "stock": 5})

	// SKUs are canonicalized to uppercase, so lookups are case-insensitive.
	rec := doJSON("POST", "/v1/orders", orderPayload(item("coffee-1kg", 3), item("TEA-250G", 2)), token)
	s.Require().Equal(201, rec.Code)

	order := decodeData[orderapi.Response](s.T(), rec)
	s.NotEqual(uuid.Nil, order.ID)
	s.Equal("pending", order.Status)
	s.Equal(int64(3*15000+2*8000), order.TotalPrice)
	s.Len(order.Items, 2)

	byProduct := make(map[uuid.UUID]orderapi.ItemResponse, len(order.Items))
	for _, orderItem := range order.Items {
		byProduct[orderItem.ProductID] = orderItem
	}
	s.Equal("Coffee", byProduct[coffee.ID].ProductName)
	s.Equal("COFFEE-1KG", byProduct[coffee.ID].SKU)
	s.Equal(int32(3), byProduct[coffee.ID].Quantity)
	s.Equal(int64(15000), byProduct[coffee.ID].UnitPrice)
	s.Equal(int64(45000), byProduct[coffee.ID].Subtotal)
	s.Equal("TEA-250G", byProduct[tea.ID].SKU, "the stored sku is the canonical uppercase form")

	// Stock is reserved immediately: 10-3=7 and 5-2=3.
	s.Equal(int32(7), getStock(s.T(), coffee.ID))
	s.Equal(int32(3), getStock(s.T(), tea.ID))

	s.Equal(1, countRows(s.T(), "orders"))
	s.Equal(2, countRows(s.T(), "order_items"))
}

func (s *orderSuite) TestCreateOrder_InsufficientStockLeavesStockUntouched() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"sku": "SCARCE-1", "name": "Scarce", "price": 10000, "stock": 2})

	rec := doJSON("POST", "/v1/orders", orderPayload(item("scarce-1", 5)), token)

	s.Require().Equal(409, rec.Code)
	errBody := decodeError(s.T(), rec)
	s.Equal("INSUFFICIENT_STOCK", errBody.Error.Code)
	s.Equal(float64(5), errBody.Error.Details["requested"])
	s.Equal(float64(2), errBody.Error.Details["available"])
	s.Equal("SCARCE-1", errBody.Error.Details["sku"])

	s.Equal(int32(2), getStock(s.T(), product.ID), "stock must remain unchanged")
	s.Equal(0, countRows(s.T(), "orders"), "no order must be created")
	s.Equal(0, countRows(s.T(), "order_items"))
}

func (s *orderSuite) TestCreateOrder_MultiItemRollsBackWhenOneItemIsShort() {
	token := registerUser(s.T()).Token
	plenty := createProduct(s.T(), token, map[string]any{"name": "Plenty", "price": 1000, "stock": 5})
	scarce := createProduct(s.T(), token, map[string]any{"name": "Scarce", "price": 2000, "stock": 1})

	rec := doJSON("POST", "/v1/orders", orderPayload(item(plenty.SKU, 2), item(scarce.SKU, 2)), token)

	s.Require().Equal(409, rec.Code)
	s.Equal("INSUFFICIENT_STOCK", decodeError(s.T(), rec).Error.Code)
	s.Equal(int32(5), getStock(s.T(), plenty.ID), "fulfillable item must not be reserved when the order fails")
	s.Equal(int32(1), getStock(s.T(), scarce.ID))
	s.Equal(0, countRows(s.T(), "orders"))
	s.Equal(0, countRows(s.T(), "order_items"))
}

func (s *orderSuite) TestCreateOrder_UnknownSKU() {
	token := registerUser(s.T()).Token

	rec := doJSON("POST", "/v1/orders", orderPayload(item("SKU-DOES-NOT-EXIST", 1)), token)

	s.Require().Equal(404, rec.Code)
	errBody := decodeError(s.T(), rec)
	s.Equal("PRODUCT_NOT_FOUND", errBody.Error.Code)
	s.Equal("SKU-DOES-NOT-EXIST", errBody.Error.Details["sku"])
	s.Equal(0, countRows(s.T(), "orders"))
}

func (s *orderSuite) TestCreateOrder_Validation() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"name": "Kopi", "price": 1000, "stock": 10})

	testCases := map[string]map[string]any{
		"empty items":       orderPayload(),
		"missing sku":       orderPayload(map[string]any{"quantity": 1}),
		"blank sku":         orderPayload(item("   ", 1)),
		"zero quantity":     orderPayload(item(product.SKU, 0)),
		"negative quantity": orderPayload(item(product.SKU, -1)),
		"duplicate sku":     orderPayload(item(product.SKU, 1), item(strings.ToLower(product.SKU), 2)),
	}

	for name, payload := range testCases {
		s.Run(name, func() {
			rec := doJSON("POST", "/v1/orders", payload, token)
			s.Equal(400, rec.Code)
			s.Equal("VALIDATION_ERROR", decodeError(s.T(), rec).Error.Code)
		})
	}

	s.Equal(int32(10), getStock(s.T(), product.ID))
	s.Equal(0, countRows(s.T(), "orders"))
}

func (s *orderSuite) TestCreateOrder_RequiresAuth() {
	product := createProduct(s.T(), registerUser(s.T()).Token, map[string]any{"name": "Kopi", "price": 1000, "stock": 10})

	rec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), "")

	s.Equal(401, rec.Code)
	s.Equal(int32(10), getStock(s.T(), product.ID))
}

// TestCreateOrder_RejectsTotalsThatOverflowInt64 pins the boundary of the
// total-price arithmetic: a single unit at the maximum price still fits, a
// second unit must be rejected as invalid input instead of overflowing into
// a 500 from the database constraint.
func (s *orderSuite) TestCreateOrder_RejectsTotalsThatOverflowInt64() {
	token := registerUser(s.T()).Token
	const maxPrice = int64(1<<63 - 1)
	product := createProduct(s.T(), token, map[string]any{"name": "Priceless", "price": maxPrice, "stock": 10})

	rec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), token)
	s.Require().Equal(201, rec.Code, "a single unit at the maximum price still fits int64")

	rec = doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 2)), token)
	s.Require().Equal(400, rec.Code)
	s.Equal("VALIDATION_ERROR", decodeError(s.T(), rec).Error.Code)
	s.Equal(int32(9), getStock(s.T(), product.ID), "rejected order must not reserve stock")
	s.Equal(1, countRows(s.T(), "orders"), "rejected order must not be persisted")
}

func (s *orderSuite) TestCreateOrder_RejectsTooManyItems() {
	token := registerUser(s.T()).Token

	items := make([]map[string]any, 101)
	for i := range items {
		items[i] = item(fmt.Sprintf("SKU-%d", i), 1)
	}

	rec := doJSON("POST", "/v1/orders", orderPayload(items...), token)

	s.Require().Equal(400, rec.Code)
	errBody := decodeError(s.T(), rec)
	s.Equal("VALIDATION_ERROR", errBody.Error.Code)
	s.Equal("max", errBody.Error.Details["items"], "the binding must reject more than 100 lines")
	s.Equal(0, countRows(s.T(), "orders"))
}

// TestCreateOrder_ConcurrentReservationOfLastItem fires many simultaneous
// orders for a product with a single unit of stock and asserts that exactly
// one order wins while the others fail without corrupting stock. Rounds start
// behind a barrier: without the FOR UPDATE row lock two requests can read the
// same stock and hit the stock CHECK constraint (a 500), which a single
// unsynchronised round only catches about two thirds of the time.
func (s *orderSuite) TestCreateOrder_ConcurrentReservationOfLastItem() {
	token := registerUser(s.T()).Token

	const rounds = 5
	const attempts = 10

	for round := range rounds {
		product := createProduct(s.T(), token, map[string]any{"name": "Last One", "price": 1000, "stock": 1})

		start := make(chan struct{})
		ready := make(chan struct{}, attempts)
		codes := make(chan int, attempts)
		var wg sync.WaitGroup

		for range attempts {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ready <- struct{}{}
				<-start
				rec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), token)
				codes <- rec.Code
			}()
		}

		for range attempts {
			<-ready
		}
		close(start)
		wg.Wait()
		close(codes)

		created, conflicts := 0, 0
		for code := range codes {
			switch code {
			case 201:
				created++
			case 409:
				conflicts++
			default:
				s.Fail("round %d: unexpected response code %d", round, code)
			}
		}

		s.Equal(1, created, "round %d: exactly one order must win the last item", round)
		s.Equal(attempts-1, conflicts, "round %d: all other attempts must fail with insufficient stock", round)
		s.Equal(int32(0), getStock(s.T(), product.ID), "round %d: stock must not go negative", round)
		s.Equal(round+1, countRows(s.T(), "orders"), "round %d: exactly one order row per round", round)
	}
}

// TestCreateOrder_ConcurrentWithCancelOnSameProducts races new orders against
// cancellations of existing orders that contain the same two products, with
// mixed request order. Creates lock products in id order and cancels restore
// them in product_id order, so the operations must interleave without
// deadlocks and without losing a stock change: the final stock must equal the
// initial stock minus the successful orders plus the completed restores.
func (s *orderSuite) TestCreateOrder_ConcurrentWithCancelOnSameProducts() {

	token := registerUser(s.T()).Token
	first := createProduct(s.T(), token, map[string]any{"name": "Race A", "price": 1000, "stock": 10})
	second := createProduct(s.T(), token, map[string]any{"name": "Race B", "price": 2000, "stock": 10})

	const pendingOrders = 6
	const newOrders = 8
	const initialStock = int32(10)

	orderIDs := make([]uuid.UUID, 0, pendingOrders)
	for range pendingOrders {
		rec := doJSON("POST", "/v1/orders", orderPayload(item(first.SKU, 1), item(second.SKU, 1)), token)
		s.Require().Equal(201, rec.Code)
		orderIDs = append(orderIDs, decodeData[orderapi.Response](s.T(), rec).ID)
	}

	start := make(chan struct{})
	cancelCodes := make(chan int, pendingOrders)
	createCodes := make(chan int, newOrders)
	var wg sync.WaitGroup

	for _, orderID := range orderIDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := doJSON("POST", "/v1/orders/"+orderID.String()+"/cancel", nil, token)
			cancelCodes <- rec.Code
		}()
	}

	for i := range newOrders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			items := []map[string]any{item(first.SKU, 1), item(second.SKU, 1)}
			if i%2 == 1 {
				// Mixed request order stresses the lock ordering against the
				// cancel restore order.
				items = []map[string]any{item(second.SKU, 1), item(first.SKU, 1)}
			}
			rec := doJSON("POST", "/v1/orders", orderPayload(items...), token)
			createCodes <- rec.Code
		}()
	}

	close(start)
	wg.Wait()
	close(cancelCodes)
	close(createCodes)

	cancelled := 0
	for code := range cancelCodes {
		s.Equal(200, code, "every first-time cancel must succeed")
		if code == 200 {
			cancelled++
		}
	}

	created, conflicts := 0, 0
	for code := range createCodes {
		switch code {
		case 201:
			created++
		case 409:
			conflicts++
		default:
			s.Fail("unexpected create response code", "code", code)
		}
	}

	s.Equal(pendingOrders, cancelled, "all pending orders must be cancelled")
	s.Equal(newOrders, created+conflicts, "every new order must succeed or conflict cleanly")

	// Each successful create removed one unit and each completed cancel
	// restored one, for both products.
	wantStock := initialStock - pendingOrders + int32(cancelled) - int32(created)
	s.Equal(wantStock, getStock(s.T(), first.ID), "stock must balance for the first product")
	s.Equal(wantStock, getStock(s.T(), second.ID), "stock must balance for the second product")

	s.Equal(pendingOrders+created, countRows(s.T(), "orders"))
	s.Equal(2*(pendingOrders+created), countRows(s.T(), "order_items"))
}
