package integration

import (
	"github.com/google/uuid"

	productapi "github.com/benebobaa/simple-order-service/internal/product"
)

func (s *productSuite) TestGetProduct() {
	product := createProduct(s.T(), registerUser(s.T()).Token, map[string]any{
		"name": "Green Tea", "price": 25000, "stock": 5,
	})

	rec := doJSON("GET", "/v1/products/"+product.ID.String(), nil, "")
	s.Require().Equal(200, rec.Code)
	got := decodeData[productapi.Response](s.T(), rec)
	s.Equal(product.ID, got.ID)
	s.Equal(product.SKU, got.SKU)
	s.Equal(product.Name, got.Name)
}

func (s *productSuite) TestGetProduct_NotFoundAndMalformedID() {
	notFound := doJSON("GET", "/v1/products/"+uuid.NewString(), nil, "")
	s.Require().Equal(404, notFound.Code)
	s.Equal("PRODUCT_NOT_FOUND", decodeError(s.T(), notFound).Error.Code)

	malformed := doJSON("GET", "/v1/products/not-a-uuid", nil, "")
	s.Equal(400, malformed.Code)
	s.Equal("VALIDATION_ERROR", decodeError(s.T(), malformed).Error.Code)
}
