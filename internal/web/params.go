package web

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/benebobaa/simple-order-service/internal/apperr"
)

const (
	defaultLimit int32 = 20
	maxLimit     int32 = 100
)

// PaginationFromQuery reads limit/offset query parameters with lenient
// clamping: invalid or negative values fall back to defaults.
func PaginationFromQuery(c *gin.Context) (limit, offset int32) {
	limit = defaultLimit
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 32); err == nil && v > 0 {
			limit = min(int32(v), maxLimit)
		}
	}
	if raw := c.Query("offset"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 32); err == nil && v >= 0 {
			offset = int32(v)
		}
	}
	return limit, offset
}

// ListMeta describes the pagination state of a list response.
type ListMeta struct {
	Total  int64 `json:"total"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// ListResponse is the shared paginated list envelope.
type ListResponse[T any] struct {
	Data []T      `json:"data"`
	Meta ListMeta `json:"meta"`
}

// ParseUUIDParam parses a UUID path parameter, aborting with a 400 response
// when it is malformed.
func ParseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		AbortWithError(c, apperr.Validation("invalid path parameter", map[string]any{name: "invalid_uuid"}))
		return uuid.Nil, false
	}
	return id, true
}
