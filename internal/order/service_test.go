package order

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/benebobaa/simple-order-service/internal/apperr"
)

func TestNormalizeItems(t *testing.T) {
	testCases := map[string]struct {
		items   []ItemInput
		want    []ItemInput
		wantErr bool
	}{
		"single valid item": {
			items: []ItemInput{{SKU: "kopi-1kg", Quantity: 1}},
			want:  []ItemInput{{SKU: "KOPI-1KG", Quantity: 1}},
		},
		"multiple valid items": {
			items: []ItemInput{{SKU: "KOPI-1KG", Quantity: 2}, {SKU: "tea-250g", Quantity: 5}},
			want:  []ItemInput{{SKU: "KOPI-1KG", Quantity: 2}, {SKU: "TEA-250G", Quantity: 5}},
		},
		"no items": {
			items:   nil,
			wantErr: true,
		},
		"empty sku": {
			items:   []ItemInput{{SKU: "   ", Quantity: 1}},
			wantErr: true,
		},
		"zero quantity": {
			items:   []ItemInput{{SKU: "KOPI-1KG", Quantity: 0}},
			wantErr: true,
		},
		"negative quantity": {
			items:   []ItemInput{{SKU: "KOPI-1KG", Quantity: -3}},
			wantErr: true,
		},
		"duplicate sku ignores case": {
			items:   []ItemInput{{SKU: "KOPI-1KG", Quantity: 1}, {SKU: "kopi-1kg", Quantity: 1}},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			normalized, err := normalizeItems(tc.items)
			if !tc.wantErr {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, normalized)
				return
			}

			var appErr *apperr.Error
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, apperr.CodeValidation, appErr.Code)
		})
	}
}
