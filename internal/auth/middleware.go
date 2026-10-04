package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/benebobaa/simple-order-service/internal/apperr"
	"github.com/benebobaa/simple-order-service/internal/web"
)

const ctxUserIDKey = "auth.user_id"

// RequireAuth rejects requests without a valid bearer token and stores the
// authenticated user ID in the gin context.
func RequireAuth(issuer TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c)
		if !ok {
			web.AbortWithError(c, apperr.Unauthorized("missing bearer token"))
			return
		}

		claims, err := issuer.Parse(token)
		if err != nil {
			web.AbortWithError(c, apperr.Unauthorized("invalid or expired token"))
			return
		}

		c.Set(ctxUserIDKey, claims.UserID)
		c.Next()
	}
}

func bearerToken(c *gin.Context) (string, bool) {
	const prefix = "Bearer "
	header := c.GetHeader("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

// UserIDFromContext returns the authenticated user ID. It must only be called
// on routes protected by RequireAuth.
func UserIDFromContext(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ctxUserIDKey)
	id, _ := v.(uuid.UUID)
	return id
}
