package integration

import (
	"sync"

	"github.com/google/uuid"

	orderapi "github.com/benebobaa/simple-order-service/internal/order"
)

func (s *orderSuite) TestCancelOrder_RestoresReservedStock() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"name": "Kopi", "price": 15000, "stock": 10})

	createRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 3)), token)
	s.Require().Equal(201, createRec.Code)
	created := decodeData[orderapi.Response](s.T(), createRec)
	s.Equal(int32(7), getStock(s.T(), product.ID))

	cancelRec := doJSON("POST", "/v1/orders/"+created.ID.String()+"/cancel", nil, token)
	s.Require().Equal(200, cancelRec.Code)

	cancelled := decodeData[orderapi.Response](s.T(), cancelRec)
	s.Equal("cancelled", cancelled.Status)
	s.Require().NotNil(cancelled.CancelledAt)

	s.Equal(int32(10), getStock(s.T(), product.ID), "reserved stock must be returned")
}

func (s *orderSuite) TestCancelOrder_AlreadyCancelledConflicts() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"name": "Kopi", "price": 15000, "stock": 10})

	createRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 3)), token)
	s.Require().Equal(201, createRec.Code)
	created := decodeData[orderapi.Response](s.T(), createRec)

	first := doJSON("POST", "/v1/orders/"+created.ID.String()+"/cancel", nil, token)
	s.Require().Equal(200, first.Code)
	s.Equal(int32(10), getStock(s.T(), product.ID))

	second := doJSON("POST", "/v1/orders/"+created.ID.String()+"/cancel", nil, token)
	s.Require().Equal(409, second.Code, "cancelling twice must conflict")
	s.Equal("ORDER_ALREADY_CANCELLED", decodeError(s.T(), second).Error.Code)

	s.Equal(int32(10), getStock(s.T(), product.ID), "stock must not be returned twice")

	// The order is still cancelled, and the failed second attempt changed nothing.
	getRec := doJSON("GET", "/v1/orders/"+created.ID.String(), nil, token)
	s.Require().Equal(200, getRec.Code)
	s.Equal("cancelled", decodeData[orderapi.Response](s.T(), getRec).Status)
}

func (s *orderSuite) TestCancelOrder_ScopedToOwner() {
	ownerToken := registerUser(s.T()).Token
	otherToken := registerUser(s.T()).Token
	product := createProduct(s.T(), ownerToken, map[string]any{"name": "Kopi", "price": 15000, "stock": 10})

	createRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 3)), ownerToken)
	s.Require().Equal(201, createRec.Code)
	created := decodeData[orderapi.Response](s.T(), createRec)

	otherCancel := doJSON("POST", "/v1/orders/"+created.ID.String()+"/cancel", nil, otherToken)
	s.Require().Equal(404, otherCancel.Code)
	s.Equal("ORDER_NOT_FOUND", decodeError(s.T(), otherCancel).Error.Code)

	// The order is still pending and stock still reserved.
	getRec := doJSON("GET", "/v1/orders/"+created.ID.String(), nil, ownerToken)
	stillPending := decodeData[orderapi.Response](s.T(), getRec)
	s.Equal("pending", stillPending.Status)
	s.Equal(int32(7), getStock(s.T(), product.ID))

	unknown := doJSON("POST", "/v1/orders/"+uuid.NewString()+"/cancel", nil, ownerToken)
	s.Equal(404, unknown.Code)

	malformed := doJSON("POST", "/v1/orders/not-a-uuid/cancel", nil, ownerToken)
	s.Equal(400, malformed.Code)

	noAuth := doJSON("POST", "/v1/orders/"+created.ID.String()+"/cancel", nil, "")
	s.Equal(401, noAuth.Code)
}

func (s *orderSuite) TestCancelOrder_AppearsInCancelledFilter() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"name": "Kopi", "price": 15000, "stock": 10})

	createRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), token)
	s.Require().Equal(201, createRec.Code)
	created := decodeData[orderapi.Response](s.T(), createRec)

	cancelRec := doJSON("POST", "/v1/orders/"+created.ID.String()+"/cancel", nil, token)
	s.Require().Equal(200, cancelRec.Code)

	listed := doJSON("GET", "/v1/orders?status=cancelled", nil, token)
	s.Require().Equal(200, listed.Code)
	env := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), listed)
	s.Require().Len(env.Data, 1)
	s.Equal(created.ID, env.Data[0].ID)
	s.Equal("cancelled", env.Data[0].Status)
	s.Require().NotNil(env.Data[0].CancelledAt)

	pending := doJSON("GET", "/v1/orders?status=pending", nil, token)
	pendingEnv := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), pending)
	s.Empty(pendingEnv.Data)
}

// TestCancelOrder_ConcurrentCancelsRestoreStockOnce races two cancel requests
// for the same order: exactly one wins (200) while the other conflicts (409),
// and stock is restored exactly once.
func (s *orderSuite) TestCancelOrder_ConcurrentCancelsRestoreStockOnce() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"name": "Kopi", "price": 15000, "stock": 10})

	createRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 4)), token)
	s.Require().Equal(201, createRec.Code)
	created := decodeData[orderapi.Response](s.T(), createRec)
	s.Require().Equal(int32(6), getStock(s.T(), product.ID))

	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := doJSON("POST", "/v1/orders/"+created.ID.String()+"/cancel", nil, token)
			codes <- rec.Code
		}()
	}
	wg.Wait()
	close(codes)

	won, conflicted := 0, 0
	for code := range codes {
		switch code {
		case 200:
			won++
		case 409:
			conflicted++
		default:
			s.Fail("unexpected cancel response code", "code", code)
		}
	}
	s.Equal(1, won, "exactly one cancel must win the transition")
	s.Equal(1, conflicted, "the losing cancel must conflict")
	s.Equal(int32(10), getStock(s.T(), product.ID), "stock must be restored exactly once")
}
