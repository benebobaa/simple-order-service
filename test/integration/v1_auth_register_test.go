package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/benebobaa/simple-order-service/internal/auth"
)

type authSuite struct {
	baseSuite
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(authSuite))
}

func (s *authSuite) TestRegister_ReturnsUserAndUsableToken() {
	rec := doJSON("POST", "/v1/auth/register", map[string]any{
		"name":     "Bene",
		"email":    "Bene@Example.com",
		"password": "s3cret-pass",
	}, "")
	s.Require().Equal(201, rec.Code)

	resp := decodeData[auth.Response](s.T(), rec)
	s.NotEmpty(resp.Token)
	s.Equal("Bearer", resp.TokenType)
	s.Equal("bene@example.com", resp.User.Email, "email should be normalized to lowercase")
	s.NotEqual(uuid.Nil, resp.User.ID)
	s.WithinDuration(time.Now().Add(time.Hour), resp.ExpiresAt, time.Minute)

	meRec := doJSON("GET", "/v1/auth/me", nil, resp.Token)
	s.Require().Equal(200, meRec.Code)
	me := decodeData[auth.UserResponse](s.T(), meRec)
	s.Equal(resp.User.ID, me.ID)
	s.NotContains(meRec.Body.String(), "password")
}

func (s *authSuite) TestRegister_DuplicateEmailConflicts() {
	registerUserWithEmail(s.T(), "dup@example.com")

	rec := doJSON("POST", "/v1/auth/register", map[string]any{
		"name":     "Someone Else",
		"email":    "DUP@example.com",
		"password": "s3cret-pass",
	}, "")

	s.Require().Equal(409, rec.Code)
	s.Equal("EMAIL_ALREADY_EXISTS", decodeError(s.T(), rec).Error.Code)
}

func (s *authSuite) TestRegister_Validation() {
	testCases := map[string]map[string]any{
		"missing name":     {"email": "a@example.com", "password": "s3cret-pass"},
		"blank name":       {"name": "   ", "email": "a@example.com", "password": "s3cret-pass"},
		"invalid email":    {"name": "A", "email": "not-an-email", "password": "s3cret-pass"},
		"short password":   {"name": "A", "email": "a@example.com", "password": "short"},
		"missing password": {"name": "A", "email": "a@example.com"},
	}

	for name, payload := range testCases {
		s.Run(name, func() {
			rec := doJSON("POST", "/v1/auth/register", payload, "")
			s.Equal(400, rec.Code)
			s.Equal("VALIDATION_ERROR", decodeError(s.T(), rec).Error.Code)
		})
	}
}
