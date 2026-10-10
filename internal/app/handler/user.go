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

type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{s: s}
}

// Register godoc
// @Summary Зарегистрировать пользователя
// @Tags users
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Email и пароль"
// @Success 201 {object} serializer.User
// @Failure 400 "Некорректный запрос"
// @Failure 409 "Email уже занят"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/users [post]
func (h *UserHandler) Register(c *gin.Context) {
	var request RegisterRequest
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

// Authenticate godoc
// @Summary Вход пользователя (пока не реализован)
// @Tags users
// @Failure 501 "Не реализовано"
// @Router /api/users/login [post]
func (h *UserHandler) Authenticate(c *gin.Context) {
	_ = h.s.Authenticate(c.Request.Context())
	c.Status(http.StatusNotImplemented)
}

// Logout godoc
// @Summary Выход пользователя (пока не реализован)
// @Tags users
// @Failure 501 "Не реализовано"
// @Router /api/users/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	_ = h.s.Logout(c.Request.Context())
	c.Status(http.StatusNotImplemented)
}
