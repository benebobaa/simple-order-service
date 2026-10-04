// Package web contains HTTP helpers shared by all feature modules: the JSON
// response envelopes, request binding and validation, pagination, request IDs
// and request logging. It intentionally knows nothing about any specific
// feature.
package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/benebobaa/simple-order-service/internal/apperr"
)

// DataResponse is the standard success envelope: {"data": ...}. List
// endpoints use ListResponse, which adds pagination metadata.
type DataResponse struct {
	Data any `json:"data"`
}

type errorResponse struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// Respond writes the standard success envelope with the given HTTP status.
func Respond(c *gin.Context, status int, data any) {
	c.JSON(status, DataResponse{Data: data})
}

// AbortWithError writes a structured error response. Unknown errors are
// logged and reported as 500 without leaking internals.
func AbortWithError(c *gin.Context, err error) {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		c.AbortWithStatusJSON(appErr.Status, errorResponse{Error: errorPayload{
			Code:      appErr.Code,
			Message:   appErr.Message,
			Details:   appErr.Details,
			RequestID: requestIDFrom(c),
		}})
		return
	}

	loggerFrom(c).Error("unhandled error",
		"error", err,
		"request_id", requestIDFrom(c),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)
	c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse{Error: errorPayload{
		Code:      "INTERNAL_ERROR",
		Message:   "internal server error",
		RequestID: requestIDFrom(c),
	}})
}

// BindJSON binds and validates a JSON request body, aborting with a 400
// response when the body is missing or invalid. It returns false when the
// request must not be processed further.
func BindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		AbortWithError(c, apperr.Validation("request body validation failed", validationDetails(err)))
		return false
	}
	return true
}

func validationDetails(err error) map[string]any {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make(map[string]any, len(validationErrs))
		for _, fieldErr := range validationErrs {
			details[strings.ToLower(fieldErr.Field())] = fieldErr.Tag()
		}
		return details
	}
	return map[string]any{"body": err.Error()}
}
