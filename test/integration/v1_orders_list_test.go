package integration

import (
	"fmt"

	"github.com/google/uuid"

	orderapi "github.com/benebobaa/simple-order-service/internal/order"
)

func (s *orderSuite) TestListOrders_ScopedToOwnerWithStatusFilter() {
	ownerToken := registerUser(s.T()).Token
	otherToken := registerUser(s.T()).Token
	product := createProduct(s.T(), ownerToken, map[string]any{"name": "Kopi", "price": 1000, "stock": 100})

	for range 3 {
		rec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), ownerToken)
		s.Require().Equal(201, rec.Code)
	}
	otherRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), otherToken)
	s.Require().Equal(201, otherRec.Code)

	ownerList := doJSON("GET", "/v1/orders", nil, ownerToken)
	s.Require().Equal(200, ownerList.Code)
	ownerEnv := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), ownerList)
	s.Len(ownerEnv.Data, 3)
	s.Equal(int64(3), ownerEnv.Meta.Total)

	otherList := doJSON("GET", "/v1/orders", nil, otherToken)
	otherEnv := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), otherList)
	s.Len(otherEnv.Data, 1, "users must only see their own orders")
	s.Equal(int64(1), otherEnv.Meta.Total)

	pendingList := doJSON("GET", "/v1/orders?status=pending", nil, ownerToken)
	pendingEnv := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), pendingList)
	s.Len(pendingEnv.Data, 3)
	s.Equal(int64(3), pendingEnv.Meta.Total)

	cancelledList := doJSON("GET", "/v1/orders?status=cancelled", nil, ownerToken)
	cancelledEnv := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), cancelledList)
	s.Empty(cancelledEnv.Data)
	s.Equal(int64(0), cancelledEnv.Meta.Total)

	invalidStatus := doJSON("GET", "/v1/orders?status=shipped", nil, ownerToken)
	s.Equal(400, invalidStatus.Code)
	s.Equal("VALIDATION_ERROR", decodeError(s.T(), invalidStatus).Error.Code)

	noAuth := doJSON("GET", "/v1/orders", nil, "")
	s.Equal(401, noAuth.Code)
}

func (s *orderSuite) TestListOrders_Pagination() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"name": "Kopi", "price": 1000, "stock": 100})

	for range 3 {
		rec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), token)
		s.Require().Equal(201, rec.Code)
	}

	firstPage := doJSON("GET", "/v1/orders?limit=2", nil, token)
	s.Require().Equal(200, firstPage.Code)
	page1 := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), firstPage)
	s.Len(page1.Data, 2)
	s.Equal(int64(3), page1.Meta.Total)
	s.Equal(int32(2), page1.Meta.Limit)

	secondPage := doJSON("GET", "/v1/orders?limit=2&offset=2", nil, token)
	page2 := decodeBody[listEnvelope[orderapi.SummaryResponse]](s.T(), secondPage)
	s.Len(page2.Data, 1)
	s.Equal(int64(3), page2.Meta.Total)

	seen := make(map[uuid.UUID]bool)
	for _, order := range append(page1.Data, page2.Data...) {
		if seen[order.ID] {
			s.Fail(fmt.Sprintf("order %s returned twice across pages", order.ID))
		}
		seen[order.ID] = true
	}
	s.Len(seen, 3)
}
