package integration

import (
	"fmt"

	"github.com/google/uuid"

	productapi "github.com/benebobaa/simple-order-service/internal/product"
)

func (s *productSuite) TestListProducts_PaginationAndTotal() {
	token := registerUser(s.T()).Token
	created := make(map[uuid.UUID]bool)
	for i := range 3 {
		product := createProduct(s.T(), token, map[string]any{
			"name": fmt.Sprintf("Product %d", i), "price": 1000, "stock": 10,
		})
		created[product.ID] = true
	}

	firstPage := doJSON("GET", "/v1/products?limit=2", nil, "")
	s.Require().Equal(200, firstPage.Code)
	page1 := decodeBody[listEnvelope[productapi.Response]](s.T(), firstPage)
	s.Len(page1.Data, 2)
	s.Equal(int64(3), page1.Meta.Total)
	s.Equal(int32(2), page1.Meta.Limit)

	secondPage := doJSON("GET", "/v1/products?limit=2&offset=2", nil, "")
	s.Require().Equal(200, secondPage.Code)
	page2 := decodeBody[listEnvelope[productapi.Response]](s.T(), secondPage)
	s.Len(page2.Data, 1)
	s.Equal(int64(3), page2.Meta.Total)
	s.Equal(int32(2), page2.Meta.Offset)

	seen := make(map[uuid.UUID]bool)
	for _, product := range append(page1.Data, page2.Data...) {
		if !created[product.ID] {
			s.Fail("unexpected product in list", product.ID)
		}
		if seen[product.ID] {
			s.Fail("product returned twice across pages", product.ID)
		}
		seen[product.ID] = true
	}
	s.Len(seen, 3)
}
