package handler

import (
	"errors"
	"net/http"

	"github.com/Juuwe/data-migration-backend/internal/app/serializer"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	s *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{s: s}
}

func (h *UserHandler) Register(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := bindJSON(c, &request); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	user, err := h.s.Register(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrInvalidEmail), errors.Is(err, service.ErrInvalidPassword):
			status = http.StatusBadRequest
		case errors.Is(err, ds.ErrUserAlreadyExists):
			status = http.StatusConflict
		}
		c.Status(status)
		return
	}

	c.JSON(http.StatusCreated, serializer.NewUser(user))
}

func (h *UserHandler) Authenticate(c *gin.Context) {
	_ = h.s.Authenticate(c.Request.Context())
	c.Status(http.StatusNotImplemented)
}

func (h *UserHandler) Logout(c *gin.Context) {
	_ = h.s.Logout(c.Request.Context())
	c.Status(http.StatusNotImplemented)
}
