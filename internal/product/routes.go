package product

import (
	"github.com/gin-gonic/gin"

	"github.com/benebobaa/simple-order-service/internal/auth"
)

// RoutesDeps carries what the product routes need.
type RoutesDeps struct {
	Service     *Service
	TokenIssuer auth.TokenIssuer
}

// RegisterRoutes mounts the product endpoints on the given group. Reads are
// public; create and update require authentication.
func RegisterRoutes(v1 *gin.RouterGroup, deps RoutesDeps) {
	h := newHandler(deps.Service)

	group := v1.Group("/products")
	group.GET("", h.list)
	group.GET("/:id", h.get)
	group.POST("", auth.RequireAuth(deps.TokenIssuer), h.create)
	group.PUT("/:id", auth.RequireAuth(deps.TokenIssuer), h.update)
}
