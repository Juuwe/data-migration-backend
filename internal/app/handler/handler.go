package handler

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/Juuwe/data-migration-backend/internal/app/serializer"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

type MigrationMethodHandler struct {
	s    *service.MigrationMethodService
	feed *service.FeedService
}

type PublishDraftRequest struct {
	Description string  `json:"description" binding:"required"`
	TimeInGb    float64 `json:"time_in_gb" binding:"required"`
	Reliability float64 `json:"reliability"`
}

type LikeRequest struct {
	Like *int `json:"like" enums:"0,1" binding:"required"`
}

func NewMigrationMethodHandler(s *service.MigrationMethodService, feeds ...*service.FeedService) *MigrationMethodHandler {
	var feedService *service.FeedService
	if len(feeds) > 0 {
		feedService = feeds[0]
	}
	return &MigrationMethodHandler{s: s, feed: feedService}
}

func pagination(c *gin.Context) (page, limit int) {
	page = defaultPage
	limit = defaultLimit

	if value, err := strconv.Atoi(c.Query("page")); err == nil && value > 0 {
		page = value
	}
	if value, err := strconv.Atoi(c.Query("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

// GetGrid godoc
// @Summary Список опубликованных методов
// @Description Если диапазон времени не задан или некорректен, фильтр не применяется. Bearer-токен необязателен.
// @Tags methods
// @Produce json
// @Security BearerAuth
// @Param min_time query number false "Минимальное время на 1 Гб"
// @Param max_time query number false "Максимальное время на 1 Гб"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество элементов на странице" default(10)
// @Success 200 {object} serializer.GridResponse
// @Failure 401 "Недействительный токен"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/methods [get]
func (h *MigrationMethodHandler) GetGrid(c *gin.Context) {
	ctx := c.Request.Context()
	currentUserID := auth.UserID(ctx)
	page, limit := pagination(c)
	offset := (page - 1) * limit

	minTime, errMin := strconv.ParseFloat(c.Query("min_time"), 64)
	maxTime, errMax := strconv.ParseFloat(c.Query("max_time"), 64)

	var (
		methods []serializer.MigrationMethod
		total   int64
		err     error
	)

	if errMin != nil || errMax != nil || minTime < 0 || maxTime < minTime ||
		math.IsNaN(minTime) || math.IsNaN(maxTime) || math.IsInf(minTime, 0) || math.IsInf(maxTime, 0) {
		minTime, maxTime = 0.01, 1.00
		methods, total, err = h.s.GetPublishedPage(ctx, currentUserID, limit, offset)
	} else {
		methods, total, err = h.s.GetPublishedByTimePage(ctx, minTime, maxTime, currentUserID, limit, offset)
	}

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	response := serializer.GridResponse{
		Methods: methods,
		MinTime: minTime,
		MaxTime: maxTime,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}

	c.JSON(http.StatusOK, response)
}

// GetFeedItem godoc
// @Summary Метод из ленты
// @Description Без id возвращает первый опубликованный метод; next=true с id — следующий с переходом к первому. Bearer-токен необязателен.
// @Tags methods
// @Produce json
// @Security BearerAuth
// @Param id query integer false "ID опубликованного метода"
// @Param next query boolean false "Получить следующий метод; требует id"
// @Success 200 {object} serializer.FeedResponse
// @Failure 400 "Некорректный запрос"
// @Failure 401 "Недействительный токен"
// @Failure 404 "Метод не найден"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/methods/feed [get]
func (h *MigrationMethodHandler) GetFeedItem(c *gin.Context) {
	idStr, hasID := c.GetQuery("id")
	ctx := c.Request.Context()
	currentUserID := auth.UserID(ctx)
	next := c.Query("next") == "true"

	var (
		method       serializer.MigrationMethod
		methodID     int64
		feedPosition int
		feedSize     int
		err          error
	)

	if h.feed == nil {
		// Резервный путь без Redis используется в unit-тестах.
		if !hasID {
			if next {
				c.Status(http.StatusBadRequest)
				return
			}
			method, err = h.s.GetNextPublishedAfterID(ctx, 0, currentUserID)
		} else {
			methodID, err = strconv.ParseInt(idStr, 10, 64)
			if err != nil || methodID <= 0 {
				c.Status(http.StatusBadRequest)
				return
			}
			if next {
				method, err = h.s.GetNextPublishedAfterID(ctx, methodID, currentUserID)
			} else {
				method, err = h.s.GetByID(ctx, methodID, currentUserID)
			}
		}
	} else {
		sessionID := feedSessionID(c)
		if sessionID == "" {
			c.Status(http.StatusInternalServerError)
			return
		}

		if !hasID {
			if next {
				c.Status(http.StatusBadRequest)
				return
			}
			cursor, feedErr := h.feed.First(ctx, sessionID)
			if feedErr != nil {
				writeFeedError(c, feedErr)
				return
			}
			methodID = cursor.MethodID
			feedPosition = cursor.Position
			feedSize = cursor.Size
		} else {
			methodID, err = strconv.ParseInt(idStr, 10, 64)
			if err != nil || methodID <= 0 {
				c.Status(http.StatusBadRequest)
				return
			}
			if next {
				cursor, feedErr := h.feed.Next(ctx, sessionID, methodID)
				if feedErr != nil {
					writeFeedError(c, feedErr)
					return
				}
				methodID = cursor.MethodID
				feedPosition = cursor.Position
				feedSize = cursor.Size
			}
		}

		method, err = h.s.GetByID(ctx, methodID, currentUserID)
	}

	if err != nil {
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			c.Status(http.StatusNotFound)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	isLiked, err := h.s.IsLiked(ctx, method.ID, currentUserID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, serializer.FeedResponse{
		Method: serializer.FeedMethod{
			MigrationMethod: method,
			IsLiked:         isLiked,
			FeedPosition:    feedPosition,
			FeedSize:        feedSize,
		},
	})
}

func writeFeedError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrFeedEmpty) {
		c.Status(http.StatusNotFound)
		return
	}
	c.Status(http.StatusInternalServerError)
}

// ShowAddMethodPage godoc
// @Summary Черновик текущего пользователя
// @Tags methods
// @Produce json
// @Security BearerAuth
// @Success 200 {object} serializer.DraftResponse
// @Failure 401 "Требуется действительный токен"
// @Failure 404 "Черновик не найден"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/methods/draft [get]
func (h *MigrationMethodHandler) ShowAddMethodPage(c *gin.Context) {
	draft, exists, err := h.s.GetDraft(c.Request.Context(), auth.UserID(c.Request.Context()))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	page := serializer.DraftResponse{
		Draft: draft,
	}
	c.JSON(http.StatusOK, page)
}

// CreateDraftMethod godoc
// @Summary Создать черновик
// @Tags methods
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param title formData string true "Название метода"
// @Param image formData file true "Изображение"
// @Param video formData file true "Видео"
// @Success 201 {object} serializer.DraftResponse
// @Header 201 {string} Location "Путь к черновику"
// @Failure 400 "Некорректный запрос"
// @Failure 401 "Требуется действительный токен"
// @Failure 413 "Файлы слишком большие"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/methods [post]
func (h *MigrationMethodHandler) CreateDraftMethod(c *gin.Context) {
	userID := auth.UserID(c.Request.Context())
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxUploadBytes)
	form, err := c.MultipartForm()
	if err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusBadRequest)
		return
	}
	defer form.RemoveAll()

	if len(form.Value) != 1 || len(form.Value["title"]) != 1 || len(form.File) != 2 || len(form.File["image"]) != 1 || len(form.File["video"]) != 1 {
		c.Status(http.StatusBadRequest)
		return
	}
	imageHeader := form.File["image"][0]
	videoHeader := form.File["video"][0]
	if imageHeader.Size <= 0 || videoHeader.Size <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	if imageHeader.Size > service.MaxImageBytes || videoHeader.Size > service.MaxVideoBytes {
		c.Status(http.StatusRequestEntityTooLarge)
		return
	}
	imageFile, err := imageHeader.Open()
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	defer imageFile.Close()
	videoFile, err := videoHeader.Open()
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	defer videoFile.Close()

	draft, err := h.s.CreateDraftMethod(c.Request.Context(), form.Value["title"][0], userID,
		service.MediaInput{Reader: imageFile, Size: imageHeader.Size},
		service.MediaInput{Reader: videoFile, Size: videoHeader.Size})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrDraftTitleRequired), errors.Is(err, service.ErrDraftTitleTooLong),
			errors.Is(err, service.ErrDraftAlreadyExists), errors.Is(err, service.ErrInvalidMedia):
			status = http.StatusBadRequest
		}
		c.Status(status)
		return
	}

	c.Header("Location", c.Request.URL.Path+"/draft")
	c.JSON(http.StatusCreated, serializer.DraftResponse{Draft: draft})
}

// PublishDraftMethod godoc
// @Summary Опубликовать черновик
// @Tags methods
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PublishDraftRequest true "Описание, время на 1 Гб и надежность"
// @Success 200 {object} serializer.MethodResponse
// @Failure 400 "Некорректный запрос"
// @Failure 401 "Требуется действительный токен"
// @Failure 404 "Черновик не найден"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/methods/publish [put]
func (h *MigrationMethodHandler) PublishDraftMethod(c *gin.Context) {
	userID := auth.UserID(c.Request.Context())
	var request PublishDraftRequest
	if err := bindJSON(c, &request); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	method, err := h.s.PublishDraft(c.Request.Context(), userID, request.Description, request.TimeInGb, request.Reliability)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidDescription), errors.Is(err, service.ErrInvalidTimeInGb),
			errors.Is(err, service.ErrInvalidReliability):
			c.Status(http.StatusBadRequest)
		case errors.Is(err, ds.ErrMigrationMethodNotFound):
			c.Status(http.StatusNotFound)
		default:
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, serializer.MethodResponse{Method: method})
}

// SoftDeleteMethod godoc
// @Summary Удалить свой метод
// @Tags methods
// @Security BearerAuth
// @Param id path integer true "ID метода"
// @Success 204 "Метод удален"
// @Failure 400 "Некорректный ID"
// @Failure 401 "Требуется действительный токен"
// @Failure 404 "Метод не найден"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/methods/{id} [delete]
func (h *MigrationMethodHandler) SoftDeleteMethod(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	err = h.s.DeleteMethod(c.Request.Context(), id, auth.UserID(c.Request.Context()))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			status = http.StatusNotFound
		}
		c.Status(status)
		return
	}

	c.Status(http.StatusNoContent)
}

// SetLike godoc
// @Summary Поставить или снять лайк
// @Tags methods
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "ID метода"
// @Param request body LikeRequest true "1 — поставить лайк, 0 — снять"
// @Success 200 {object} serializer.LikeResponse
// @Failure 400 "Некорректный запрос"
// @Failure 401 "Требуется действительный токен"
// @Failure 404 "Метод не найден"
// @Failure 500 "Внутренняя ошибка"
// @Router /api/methods/{id}/like [post]
func (h *MigrationMethodHandler) SetLike(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	var request LikeRequest
	if err := bindJSON(c, &request); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if request.Like == nil {
		c.Status(http.StatusBadRequest)
		return
	}

	method, err := h.s.SetLike(c.Request.Context(), id, auth.UserID(c.Request.Context()), *request.Like)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrInvalidLike):
			status = http.StatusBadRequest
		case errors.Is(err, ds.ErrMigrationMethodNotFound):
			status = http.StatusNotFound
		}
		c.Status(status)
		return
	}

	c.JSON(http.StatusOK, serializer.LikeResponse{Method: method})
}
