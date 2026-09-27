package handler

import (
	"errors"
	"net/http"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.s.Register(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case errors.Is(err, service.ErrInvalidEmail), errors.Is(err, service.ErrInvalidPassword):
			status = http.StatusBadRequest
		case errors.Is(err, ds.ErrUserAlreadyExists):
			status = http.StatusConflict
		default:
			message = "Не удалось зарегистрировать пользователя"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "email": user.Email})
}

func (h *UserHandler) Authenticate(c *gin.Context) {
	err := h.s.Authenticate(c.Request.Context())
	c.JSON(http.StatusNotImplemented, gin.H{"error": err.Error()})
}

func (h *UserHandler) Logout(c *gin.Context) {
	err := h.s.Logout(c.Request.Context())
	c.JSON(http.StatusNotImplemented, gin.H{"error": err.Error()})
}
