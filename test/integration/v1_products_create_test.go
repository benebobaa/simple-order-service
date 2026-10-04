package integration

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	productapi "github.com/benebobaa/simple-order-service/internal/product"
)

type productSuite struct {
	baseSuite
}

func TestProductSuite(t *testing.T) {
	suite.Run(t, new(productSuite))
}

func (s *productSuite) TestCreateProduct() {
	token := registerUser(s.T()).Token

	rec := doJSON("POST", "/v1/products", map[string]any{
		"sku":         "kopi-arabika-1kg",
		"name":        "Kopi Arabika",
		"description": "Single origin, 1kg",
		"price":       150000,
		"stock":       10,
	}, token)

	s.Require().Equal(201, rec.Code)
	product := decodeData[productapi.Response](s.T(), rec)
	s.NotEqual(uuid.Nil, product.ID)
	s.Equal("KOPI-ARABIKA-1KG", product.SKU, "sku must be canonicalized to uppercase")
	s.Equal("Kopi Arabika", product.Name)
	s.Equal("Single origin, 1kg", product.Description)
	s.Equal(int64(150000), product.Price)
	s.Equal(int32(10), product.Stock)
	s.False(product.CreatedAt.IsZero())
}

func (s *productSuite) TestCreateProduct_AcceptsZeroPriceAndStock() {
	product := createProduct(s.T(), registerUser(s.T()).Token, map[string]any{
		"name": "Free Sample", "price": 0, "stock": 0,
	})
	s.NotEmpty(product.SKU, "the helper must generate a sku when none is given")
	s.Equal(int64(0), product.Price)
	s.Equal(int32(0), product.Stock)
}

func (s *productSuite) TestCreateProduct_RequiresAuth() {
	rec := doJSON("POST", "/v1/products", map[string]any{
		"sku": "KOPI-1", "name": "Kopi", "price": 1000, "stock": 1,
	}, "")

	s.Equal(401, rec.Code)
	s.Equal("UNAUTHORIZED", decodeError(s.T(), rec).Error.Code)
}

func (s *productSuite) TestCreateProduct_DuplicateSKU() {
	token := registerUser(s.T()).Token
	createProduct(s.T(), token, map[string]any{"sku": "DUP-1", "name": "First", "price": 1000, "stock": 1})

	// Lowercase input must collide with the stored canonical "DUP-1".
	rec := doJSON("POST", "/v1/products", map[string]any{
		"sku": "dup-1", "name": "Second", "price": 2000, "stock": 2,
	}, token)

	s.Require().Equal(409, rec.Code)
	s.Equal("SKU_ALREADY_EXISTS", decodeError(s.T(), rec).Error.Code)
	s.Equal(1, countRows(s.T(), "products"))
}

func (s *productSuite) TestCreateProduct_Validation() {
	token := registerUser(s.T()).Token

	testCases := map[string]map[string]any{
		"missing sku":      {"name": "Kopi", "price": 1000, "stock": 1},
		"sku too short":    {"sku": "K", "name": "Kopi", "price": 1000, "stock": 1},
		"padded short sku": {"sku": "  A  ", "name": "Kopi", "price": 1000, "stock": 1},
		"missing name":     {"sku": "KOPI-1", "price": 1000, "stock": 1},
		"blank name":       {"sku": "KOPI-1", "name": "   ", "price": 1000, "stock": 1},
		"missing price":    {"sku": "KOPI-1", "name": "Kopi", "stock": 1},
		"negative price":   {"sku": "KOPI-1", "name": "Kopi", "price": -1, "stock": 1},
		"missing stock":    {"sku": "KOPI-1", "name": "Kopi", "price": 1000},
		"negative stock":   {"sku": "KOPI-1", "name": "Kopi", "price": 1000, "stock": -1},
	}

	for name, payload := range testCases {
		s.Run(name, func() {
			rec := doJSON("POST", "/v1/products", payload, token)
			s.Equal(400, rec.Code)
			s.Equal("VALIDATION_ERROR", decodeError(s.T(), rec).Error.Code)
		})
	}
}
