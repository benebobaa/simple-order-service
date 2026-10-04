package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTIssuerRoundTrip(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", time.Hour)
	userID := uuid.Must(uuid.NewV7())

	token, expiresAt, err := issuer.Generate(userID, "user@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, time.Minute)

	claims, err := issuer.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, userID.String(), claims.Subject)
}

func TestJWTIssuerRejectsExpiredToken(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", -time.Minute)

	token, _, err := issuer.Generate(uuid.Must(uuid.NewV7()), "user@example.com")
	require.NoError(t, err)

	_, err = issuer.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTIssuerRejectsWrongSecret(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", time.Hour)
	token, _, err := issuer.Generate(uuid.Must(uuid.NewV7()), "user@example.com")
	require.NoError(t, err)

	other := NewJWTIssuer("other-secret", time.Hour)
	_, err = other.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTIssuerRejectsMalformedToken(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", time.Hour)

	_, err := issuer.Parse("not-a-jwt")
	assert.ErrorIs(t, err, ErrInvalidToken)

	_, err = issuer.Parse("")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTIssuerRejectsNonHMACAlgorithm(t *testing.T) {
	claims := Claims{
		Email: "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.Must(uuid.NewV7()).String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	// Sign with a different algorithm family than the issuer expects.
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	issuer := NewJWTIssuer("test-secret", time.Hour)
	_, err = issuer.Parse(signed)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTIssuerRejectsTokenWithoutExpiry(t *testing.T) {
	claims := Claims{
		Email: "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: uuid.Must(uuid.NewV7()).String(),
			Issuer:  tokenIssuer,
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	require.NoError(t, err)

	issuer := NewJWTIssuer("test-secret", time.Hour)
	_, err = issuer.Parse(signed)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTIssuerRejectsWrongIssuer(t *testing.T) {
	claims := Claims{
		Email: "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.Must(uuid.NewV7()).String(),
			Issuer:    "some-other-service",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	require.NoError(t, err)

	issuer := NewJWTIssuer("test-secret", time.Hour)
	_, err = issuer.Parse(signed)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
