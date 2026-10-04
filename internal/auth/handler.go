package auth

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

func (h *handler) register(c *gin.Context) {
	var req registerRequest
	if !web.BindJSON(c, &req) {
		return
	}

	result, err := h.service.Register(c.Request.Context(), RegisterInput(req))
	if err != nil {
		web.AbortWithError(c, err)
		return
	}

	web.Respond(c, http.StatusCreated, newResponse(result))
}

func (h *handler) login(c *gin.Context) {
	var req loginRequest
	if !web.BindJSON(c, &req) {
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		web.AbortWithError(c, err)
		return
	}

	web.Respond(c, http.StatusOK, newResponse(result))
}

func (h *handler) me(c *gin.Context) {
	user, err := h.service.GetUser(c.Request.Context(), UserIDFromContext(c))
	if err != nil {
		web.AbortWithError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, newUserResponse(user))
}
