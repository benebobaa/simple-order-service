package order

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/benebobaa/simple-order-service/internal/auth"
	"github.com/benebobaa/simple-order-service/internal/web"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) create(c *gin.Context) {
	var req createOrderRequest
	if !web.BindJSON(c, &req) {
		return
	}

	items := make([]ItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, ItemInput(item))
	}

	detail, err := h.service.Create(c.Request.Context(), auth.UserIDFromContext(c), items)
	if err != nil {
		web.AbortWithError(c, err)
		return
	}

	web.Respond(c, http.StatusCreated, newResponse(detail))
}
