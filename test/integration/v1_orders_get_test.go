package integration

import (
	"github.com/google/uuid"

	orderapi "github.com/benebobaa/simple-order-service/internal/order"
)

func (s *orderSuite) TestGetOrder_ReturnsItems() {
	token := registerUser(s.T()).Token
	product := createProduct(s.T(), token, map[string]any{"sku": "kopi-250g", "name": "Kopi", "price": 25000, "stock": 10})

	createRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 2)), token)
	s.Require().Equal(201, createRec.Code)
	created := decodeData[orderapi.Response](s.T(), createRec)

	getRec := doJSON("GET", "/v1/orders/"+created.ID.String(), nil, token)
	s.Require().Equal(200, getRec.Code)

	order := decodeData[orderapi.Response](s.T(), getRec)
	s.Equal(created.ID, order.ID)
	s.Equal(created.UserID, order.UserID)
	s.Equal("pending", order.Status)
	s.Equal(int64(50000), order.TotalPrice)
	s.Require().Len(order.Items, 1)
	s.Equal(product.ID, order.Items[0].ProductID)
	s.Equal("KOPI-250G", order.Items[0].SKU)
	s.Equal("Kopi", order.Items[0].ProductName)
	s.Equal(int32(2), order.Items[0].Quantity)
}

func (s *orderSuite) TestGetOrder_ScopedToOwner() {
	ownerToken := registerUser(s.T()).Token
	otherToken := registerUser(s.T()).Token
	product := createProduct(s.T(), ownerToken, map[string]any{"name": "Kopi", "price": 1000, "stock": 10})

	createRec := doJSON("POST", "/v1/orders", orderPayload(item(product.SKU, 1)), ownerToken)
	s.Require().Equal(201, createRec.Code)
	created := decodeData[orderapi.Response](s.T(), createRec)

	otherUser := doJSON("GET", "/v1/orders/"+created.ID.String(), nil, otherToken)
	s.Require().Equal(404, otherUser.Code, "another user's order must not be visible")
	s.Equal("ORDER_NOT_FOUND", decodeError(s.T(), otherUser).Error.Code)

	unknown := doJSON("GET", "/v1/orders/"+uuid.NewString(), nil, ownerToken)
	s.Equal(404, unknown.Code)

	malformed := doJSON("GET", "/v1/orders/not-a-uuid", nil, ownerToken)
	s.Equal(400, malformed.Code)

	noAuth := doJSON("GET", "/v1/orders/"+created.ID.String(), nil, "")
	s.Equal(401, noAuth.Code)
}
