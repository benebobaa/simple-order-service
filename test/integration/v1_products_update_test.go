package integration

import (
	"github.com/google/uuid"

	productapi "github.com/benebobaa/simple-order-service/internal/product"
)

func (s *productSuite) TestUpdateProduct() {
	product := createProduct(s.T(), registerUser(s.T()).Token, map[string]any{
		"sku": "OLD-SKU", "name": "Old Name", "price": 1000, "stock": 3,
	})
	token := registerUser(s.T()).Token

	// The SKU is intentionally included here to prove it is ignored: the
	// business key is immutable after creation.
	rec := doJSON("PUT", "/v1/products/"+product.ID.String(), map[string]any{
		"sku": "NEW-SKU", "name": "New Name", "description": "updated", "price": 2000, "stock": 7,
	}, token)

	s.Require().Equal(200, rec.Code)
	updated := decodeData[productapi.Response](s.T(), rec)
	s.Equal("New Name", updated.Name)
	s.Equal("updated", updated.Description)
	s.Equal(int64(2000), updated.Price)
	s.Equal(int32(7), updated.Stock)
	s.Equal("OLD-SKU", updated.SKU, "the sku must be immutable")

	getRec := doJSON("GET", "/v1/products/"+product.ID.String(), nil, "")
	got := decodeData[productapi.Response](s.T(), getRec)
	s.Equal(updated.Name, got.Name)
	s.Equal(updated.Price, got.Price)
	s.Equal("OLD-SKU", got.SKU)
}

func (s *productSuite) TestUpdateProduct_NotFoundValidationAndAuth() {
	token := registerUser(s.T()).Token

	notFound := doJSON("PUT", "/v1/products/"+uuid.NewString(), map[string]any{
		"name": "X", "price": 1, "stock": 1,
	}, token)
	s.Require().Equal(404, notFound.Code)
	s.Equal("PRODUCT_NOT_FOUND", decodeError(s.T(), notFound).Error.Code)

	product := createProduct(s.T(), token, map[string]any{"name": "X", "price": 1, "stock": 1})
	invalid := doJSON("PUT", "/v1/products/"+product.ID.String(), map[string]any{
		"name": "", "price": -5, "stock": 1,
	}, token)
	s.Equal(400, invalid.Code)

	noAuth := doJSON("PUT", "/v1/products/"+product.ID.String(), map[string]any{
		"name": "Y", "price": 1, "stock": 1,
	}, "")
	s.Equal(401, noAuth.Code)
}
