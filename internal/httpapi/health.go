package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/benebobaa/simple-order-service/internal/apperr"
	"github.com/benebobaa/simple-order-service/internal/web"
)

const readinessTimeout = 2 * time.Second

// healthz reports liveness: the process is running.
func healthz(c *gin.Context) {
	web.Respond(c, http.StatusOK, gin.H{"status": "ok"})
}

// readyz reports readiness: the service's dependencies are reachable. The
// check is bounded so a hung dependency cannot hang the probe.
func readyz(check func(ctx context.Context) error, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if check == nil {
			web.Respond(c, http.StatusOK, gin.H{"status": "ready"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
		defer cancel()

		if err := check(ctx); err != nil {
			logger.Error("readiness check failed", "error", err)
			web.AbortWithError(c, apperr.New(http.StatusServiceUnavailable, "NOT_READY", "service dependencies are unavailable"))
			return
		}
		web.Respond(c, http.StatusOK, gin.H{"status": "ready"})
	}
}
