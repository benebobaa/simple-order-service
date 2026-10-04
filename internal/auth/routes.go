package auth

import (
	"github.com/gin-gonic/gin"
)

// RoutesDeps carries what the auth routes need.
type RoutesDeps struct {
	Service     *Service
	TokenIssuer TokenIssuer
}

// RegisterRoutes mounts the authentication endpoints on the given group.
func RegisterRoutes(v1 *gin.RouterGroup, deps RoutesDeps) {
	h := newHandler(deps.Service)

	group := v1.Group("/auth")
	group.POST("/register", h.register)
	group.POST("/login", h.login)
	group.GET("/me", RequireAuth(deps.TokenIssuer), h.me)
}
