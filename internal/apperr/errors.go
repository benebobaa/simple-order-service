// Package apperr defines the typed application errors shared by the service
// and HTTP layers.
package apperr

import (
	"fmt"
	"net/http"
)

// API error codes returned to clients.
const (
	CodeValidation            = "VALIDATION_ERROR"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeInvalidCredentials    = "INVALID_CREDENTIALS"
	CodeEmailExists           = "EMAIL_ALREADY_EXISTS"
	CodeSKUExists             = "SKU_ALREADY_EXISTS"
	CodeUserNotFound          = "USER_NOT_FOUND"
	CodeProductNotFound       = "PRODUCT_NOT_FOUND"
	CodeOrderNotFound         = "ORDER_NOT_FOUND"
	CodeOrderAlreadyCancelled = "ORDER_ALREADY_CANCELLED"
	CodeInsufficientStock     = "INSUFFICIENT_STOCK"
)

// Error is an application error carrying an API error code, an HTTP status
// and optional structured details.
type Error struct {
	Code    string
	Message string
	Status  int
	Details map[string]any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New builds an application error.
func New(status int, code, message string) *Error {
	return &Error{Code: code, Message: message, Status: status}
}

// Validation returns a 400 error describing an invalid request.
func Validation(message string, details map[string]any) *Error {
	return &Error{Code: CodeValidation, Message: message, Status: http.StatusBadRequest, Details: details}
}

// Unauthorized returns a 401 error for missing or invalid authentication.
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, CodeUnauthorized, message)
}

// InvalidCredentials returns the generic 401 used for failed login attempts,
// deliberately not revealing whether the email or the password was wrong.
func InvalidCredentials() *Error {
	return New(http.StatusUnauthorized, CodeInvalidCredentials, "invalid email or password")
}

// NotFound returns a 404 error for a missing resource.
func NotFound(code, message string) *Error {
	return New(http.StatusNotFound, code, message)
}

// Conflict returns a 409 error for state conflicts such as insufficient stock.
func Conflict(code, message string, details map[string]any) *Error {
	return &Error{Code: code, Message: message, Status: http.StatusConflict, Details: details}
}

// WithDetails returns a copy of the error with structured details attached.
func (e *Error) WithDetails(details map[string]any) *Error {
	copied := *e
	copied.Details = details
	return &copied
}
