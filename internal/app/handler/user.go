package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Juuwe/data-migration-backend/internal/app/serializer"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	s *service.UserService
}

type Credentials struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{s: s}
}

// Register godoc
// @Summary Зарегистрировать пользователя
// @Tags users
// @Accept json
// @Produce json
// @Param request body Credentials true "Email и пароль"
// @Success 201 {object} serializer.User
// @Failure 400 "Некорректный запрос"
// @Failure 409 "Email уже занят"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/users [post]
func (h *UserHandler) Register(c *gin.Context) {
	var request Credentials
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
// @Summary Вход пользователя
// @Tags users
// @Accept json
// @Produce json
// @Param request body Credentials true "Email и пароль"
// @Success 200 {object} serializer.TokenResponse
// @Failure 400 "Некорректный запрос"
// @Failure 401 "Неверный email или пароль"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/users/login [post]
func (h *UserHandler) Authenticate(c *gin.Context) {
	var request Credentials
	if err := bindJSON(c, &request); err != nil || strings.TrimSpace(request.Email) == "" || request.Password == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	tokens, err := h.s.Authenticate(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.Status(http.StatusUnauthorized)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	c.JSON(http.StatusOK, serializer.TokenResponse{AccessToken: tokens.AccessToken, TokenType: "Bearer", ExpiresIn: tokens.ExpiresIn, RefreshToken: tokens.RefreshToken})
}

// Refresh godoc
// @Summary Обновить пару токенов
// @Description Refresh-токен одноразовый; при успешном обмене старый токен перестает действовать.
// @Tags users
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh-токен"
// @Success 200 {object} serializer.TokenResponse
// @Failure 400 "Некорректный запрос"
// @Failure 401 "Недействительный refresh-токен"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/users/refresh [post]
func (h *UserHandler) Refresh(c *gin.Context) {
	var request RefreshRequest
	if err := bindJSON(c, &request); err != nil || request.RefreshToken == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	tokens, err := h.s.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			c.Status(http.StatusUnauthorized)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	c.JSON(http.StatusOK, serializer.TokenResponse{AccessToken: tokens.AccessToken, TokenType: "Bearer", ExpiresIn: tokens.ExpiresIn, RefreshToken: tokens.RefreshToken})
}

// Logout godoc
// @Summary Выход пользователя
// @Description Добавляет access-токен в Redis-блэклист до истечения срока и отзывает refresh-токен. После ответа удалите оба токена на клиенте.
// @Tags users
// @Accept json
// @Security BearerAuth
// @Param request body RefreshRequest true "Текущий refresh-токен"
// @Success 204 "Без тела"
// @Failure 400 "Некорректный запрос"
// @Failure 401 "Недействительный access- или refresh-токен"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/users/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	var request RefreshRequest
	if err := bindJSON(c, &request); err != nil || request.RefreshToken == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.Status(http.StatusUnauthorized)
		return
	}
	if err := h.s.Logout(c.Request.Context(), parts[1], request.RefreshToken); err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) || errors.Is(err, auth.ErrInvalidToken) {
			c.Status(http.StatusUnauthorized)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	c.Status(http.StatusNoContent)
}
