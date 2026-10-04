package web

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

const ctxLoggerKey = "web.logger"

// RequestLogger emits one structured log line per HTTP request and stores the
// logger in the request context for error reporting.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Set(ctxLoggerKey, logger)
		c.Next()

		logger.Info("http_request",
			"request_id", requestIDFrom(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

func loggerFrom(c *gin.Context) *slog.Logger {
	if v, ok := c.Get(ctxLoggerKey); ok {
		if logger, ok := v.(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}
