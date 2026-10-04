// Package httpapi builds the HTTP router and mounts the feature modules.
package httpapi

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/benebobaa/simple-order-service/internal/auth"
	"github.com/benebobaa/simple-order-service/internal/order"
	"github.com/benebobaa/simple-order-service/internal/product"
	"github.com/benebobaa/simple-order-service/internal/web"
)

// Deps carries the route dependencies for every feature module.
type Deps struct {
	Logger *slog.Logger
	// ReadyCheck reports whether the service can serve traffic (dependencies
	// reachable). It is used by the /readyz probe. When nil the probe reports
	// ready without checking dependencies.
	ReadyCheck func(ctx context.Context) error
	Auth       auth.RoutesDeps
	Products   product.RoutesDeps
	Orders     order.RoutesDeps
}

// NewRouter builds the gin engine, applies global middleware and mounts each
// feature module. All route definitions live inside their own module.
func NewRouter(deps Deps) *gin.Engine {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery(), web.RequestID(), web.RequestLogger(deps.Logger))

	r.GET("/healthz", healthz)
	r.GET("/readyz", readyz(deps.ReadyCheck, deps.Logger))

	v1 := r.Group("/v1")
	auth.RegisterRoutes(v1, deps.Auth)
	product.RegisterRoutes(v1, deps.Products)
	order.RegisterRoutes(v1, deps.Orders)

	return r
}
