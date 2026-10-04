// Package auth implements user registration, sign-in, JWT issuance and
// request authentication.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken is returned when a token is malformed, expired or signed
// with the wrong key.
var ErrInvalidToken = errors.New("invalid token")

// tokenIssuer identifies tokens minted and accepted by this service.
const tokenIssuer = "simple-order-service"

// Claims are the JWT claims embedded in an access token.
type Claims struct {
	Email  string    `json:"email"`
	UserID uuid.UUID `json:"-"`
	jwt.RegisteredClaims
}

// TokenIssuer generates and validates access tokens.
type TokenIssuer interface {
	Generate(userID uuid.UUID, email string) (token string, expiresAt time.Time, err error)
	Parse(token string) (*Claims, error)
}

// JWTIssuer is a TokenIssuer backed by HMAC-SHA256 JWTs.
type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTIssuer creates a JWT issuer. ttl must be positive.
func NewJWTIssuer(secret string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), ttl: ttl}
}

// Generate creates a signed token for the given user.
func (i *JWTIssuer) Generate(userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(i.ttl)
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse validates a token and returns its claims. Accepted tokens must be
// signed with HS256, carry an expiry and have been issued by this service.
func (i *JWTIssuer) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (any, error) {
		return i.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims.UserID = userID
	return claims, nil
}
