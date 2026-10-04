// Package httpapi builds the HTTP router and mounts the feature modules.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/benebobaa/simple-order-service/internal/auth"
	"github.com/benebobaa/simple-order-service/internal/product"
	"github.com/benebobaa/simple-order-service/internal/web"
)

// Deps carries the route dependencies for every feature module.
type Deps struct {
	Logger   *slog.Logger
	Auth     auth.RoutesDeps
	Products product.RoutesDeps
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

	r.GET("/healthz", func(c *gin.Context) {
		web.Respond(c, http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/v1")
	auth.RegisterRoutes(v1, deps.Auth)
	product.RegisterRoutes(v1, deps.Products)

	return r
}
