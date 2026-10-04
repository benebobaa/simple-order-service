package web

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader is the response/request header used to correlate a request
// across clients, logs and error responses.
const RequestIDHeader = "X-Request-ID"

const ctxRequestIDKey = "web.request_id"

// RequestID echoes a client-supplied request ID or generates one, stores it in
// the request context and returns it in the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if id == "" || len(id) > 128 {
			if generated, err := uuid.NewV7(); err == nil {
				id = generated.String()
			} else {
				id = uuid.NewString()
			}
		}

		c.Set(ctxRequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

func requestIDFrom(c *gin.Context) string {
	v, _ := c.Get(ctxRequestIDKey)
	id, _ := v.(string)
	return id
}
