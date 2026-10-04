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

func (h *handler) get(c *gin.Context) {
	id, ok := web.ParseUUIDParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.service.Get(c.Request.Context(), auth.UserIDFromContext(c), id)
	if err != nil {
		web.AbortWithError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, newResponse(detail))
}

func (h *handler) list(c *gin.Context) {
	limit, offset := web.PaginationFromQuery(c)

	var status *string
	if raw, exists := c.GetQuery("status"); exists {
		status = &raw
	}

	orders, total, err := h.service.List(c.Request.Context(), auth.UserIDFromContext(c), status, limit, offset)
	if err != nil {
		web.AbortWithError(c, err)
		return
	}

	items := make([]SummaryResponse, 0, len(orders))
	for _, order := range orders {
		items = append(items, newSummaryResponse(order))
	}

	c.JSON(http.StatusOK, web.ListResponse[SummaryResponse]{
		Data: items,
		Meta: web.ListMeta{Total: total, Limit: limit, Offset: offset},
	})
}

func (h *handler) cancel(c *gin.Context) {
	id, ok := web.ParseUUIDParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.service.Cancel(c.Request.Context(), auth.UserIDFromContext(c), id)
	if err != nil {
		web.AbortWithError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, newResponse(detail))
}
