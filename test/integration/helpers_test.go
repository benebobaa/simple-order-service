package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/benebobaa/simple-order-service/internal/auth"
)

// baseSuite truncates all tables before each test so tests are independent.
type baseSuite struct {
	suite.Suite
}

func (s *baseSuite) SetupTest() {
	_, err := testPool.Exec(context.Background(),
		"TRUNCATE order_items, orders, products, users RESTART IDENTITY CASCADE")
	s.Require().NoError(err)
}

// doJSON performs an HTTP request against the real router. token, when set,
// is sent as a bearer token.
func doJSON(method, path string, body any, token string) *httptest.ResponseRecorder {
	return doJSONWithHeaders(method, path, body, token, nil)
}

// doJSONWithHeaders performs a request with extra headers (e.g. X-Request-ID).
func doJSONWithHeaders(method, path string, body any, token string, headers map[string]string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			panic(fmt.Sprintf("marshal request body: %v", err))
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequestWithContext(context.Background(), method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	return rec
}

// decodeBody decodes a JSON response body into T.
func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
	return out
}

// dataEnvelope mirrors the standard success envelope shape.
type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

// decodeData decodes a single-resource success response ({"data": ...}).
func decodeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	return decodeBody[dataEnvelope[T]](t, rec).Data
}

// errorEnvelope mirrors the API error response shape.
type errorEnvelope struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	return decodeBody[errorEnvelope](t, rec)
}

// registerUser registers a fresh random user and returns the auth response.
func registerUser(t *testing.T) auth.Response {
	t.Helper()
	return registerUserWithEmail(t, fmt.Sprintf("user-%s@example.com", uuid.NewString()))
}

// registerUserWithEmail registers a user with the given email.
func registerUserWithEmail(t *testing.T, email string) auth.Response {
	t.Helper()
	rec := doJSON("POST", "/v1/auth/register", map[string]any{
		"name":     "Test User",
		"email":    email,
		"password": "s3cret-pass",
	}, "")
	if rec.Code != 201 {
		t.Fatalf("register user: status %d body %s", rec.Code, rec.Body.String())
	}
	return decodeData[auth.Response](t, rec)
}
