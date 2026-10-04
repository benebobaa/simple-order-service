// Package app wires the application together and runs its lifecycle.
package app

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benebobaa/simple-order-service/internal/auth"
	"github.com/benebobaa/simple-order-service/internal/config"
	"github.com/benebobaa/simple-order-service/internal/httpapi"
	"github.com/benebobaa/simple-order-service/internal/order"
	"github.com/benebobaa/simple-order-service/internal/product"
	"github.com/benebobaa/simple-order-service/internal/store"
)

// New builds the complete HTTP application: store, feature services and the
// router with all modules mounted.
func New(pool *pgxpool.Pool, cfg *config.Config, logger *slog.Logger) *gin.Engine {
	st := store.New(pool)
	issuer := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)

	return httpapi.NewRouter(httpapi.Deps{
		Logger: logger,
		Auth: auth.RoutesDeps{
			Service:     auth.NewService(st, issuer),
			TokenIssuer: issuer,
		},
		Products: product.RoutesDeps{
			Service:     product.NewService(st),
			TokenIssuer: issuer,
		},
		Orders: order.RoutesDeps{
			Service:     order.NewService(st),
			TokenIssuer: issuer,
		},
	})
}
