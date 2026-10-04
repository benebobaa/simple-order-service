package product

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/benebobaa/simple-order-service/internal/web"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) create(c *gin.Context) {
	var req createRequest
	if !web.BindJSON(c, &req) {
		return
	}

	product, err := h.service.Create(c.Request.Context(), req.input())
	if err != nil {
		web.AbortWithError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, newResponse(product))
}

func (h *handler) get(c *gin.Context) {
	id, ok := web.ParseUUIDParam(c, "id")
	if !ok {
		return
	}

	product, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		web.AbortWithError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, newResponse(product))
}

func (h *handler) list(c *gin.Context) {
	limit, offset := web.PaginationFromQuery(c)

	products, total, err := h.service.List(c.Request.Context(), limit, offset)
	if err != nil {
		web.AbortWithError(c, err)
		return
	}

	items := make([]Response, 0, len(products))
	for _, product := range products {
		items = append(items, newResponse(product))
	}

	c.JSON(http.StatusOK, web.ListResponse[Response]{
		Data: items,
		Meta: web.ListMeta{Total: total, Limit: limit, Offset: offset},
	})
}

func (h *handler) update(c *gin.Context) {
	id, ok := web.ParseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req updateRequest
	if !web.BindJSON(c, &req) {
		return
	}

	product, err := h.service.Update(c.Request.Context(), id, req.input())
	if err != nil {
		web.AbortWithError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, newResponse(product))
}
