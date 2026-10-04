package product

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/benebobaa/simple-order-service/internal/apperr"
)

func TestNormalizeInput(t *testing.T) {
	t.Run("trims the name", func(t *testing.T) {
		input, err := normalizeInput(Input{Name: "  Kopi Arabika  ", Price: 150000, Stock: 10})
		require.NoError(t, err)
		assert.Equal(t, "Kopi Arabika", input.Name)
	})

	t.Run("allows zero price and stock", func(t *testing.T) {
		input, err := normalizeInput(Input{Name: "Free Sample", Price: 0, Stock: 0})
		require.NoError(t, err)
		assert.Equal(t, int64(0), input.Price)
		assert.Equal(t, int32(0), input.Stock)
	})

	testCases := map[string]struct {
		input Input
	}{
		"blank name":     {input: Input{Name: "   ", Price: 1, Stock: 1}},
		"negative price": {input: Input{Name: "Kopi", Price: -1, Stock: 1}},
		"negative stock": {input: Input{Name: "Kopi", Price: 1, Stock: -1}},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := normalizeInput(tc.input)

			var appErr *apperr.Error
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, apperr.CodeValidation, appErr.Code)
		})
	}
}

func TestNormalizeCreateInput(t *testing.T) {
	t.Run("canonicalizes the sku", func(t *testing.T) {
		input, err := normalizeCreateInput(Input{SKU: "  kopi-1kg ", Name: "Kopi", Price: 150000, Stock: 10})
		require.NoError(t, err)
		assert.Equal(t, "KOPI-1KG", input.SKU)
	})

	testCases := map[string]string{
		"empty sku": "   ",
		"too short": "A",
	}

	for name, raw := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := normalizeCreateInput(Input{SKU: raw, Name: "Kopi", Price: 1, Stock: 1})

			var appErr *apperr.Error
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, apperr.CodeValidation, appErr.Code)
		})
	}
}
