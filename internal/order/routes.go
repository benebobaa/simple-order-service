package order

import (
	"github.com/gin-gonic/gin"

	"github.com/benebobaa/simple-order-service/internal/auth"
)

// RoutesDeps carries what the order routes need.
type RoutesDeps struct {
	Service     *Service
	TokenIssuer auth.TokenIssuer
}

// RegisterRoutes mounts the order endpoints on the given group. All order
// routes require authentication and operate on the authenticated user.
func RegisterRoutes(v1 *gin.RouterGroup, deps RoutesDeps) {
	h := newHandler(deps.Service)
	requireAuth := auth.RequireAuth(deps.TokenIssuer)

	group := v1.Group("/orders")
	group.POST("", requireAuth, h.create)
	group.GET("", requireAuth, h.list)
	group.GET("/:id", requireAuth, h.get)
	group.POST("/:id/cancel", requireAuth, h.cancel)
}
