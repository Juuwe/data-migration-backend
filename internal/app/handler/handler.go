package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

type GridPageData struct {
	Methods []service.MigrationMethodView `json:"methods"`
	MinTime float64                       `json:"min_time"`
	MaxTime float64                       `json:"max_time"`
}

type FeedPageData struct {
	Method service.MigrationMethodView `json:"method"`
}

type AddPageData struct {
	Draft  service.MigrationMethodView `json:"draft"`
	Exists bool                        `json:"exists"`
}

type MigrationMethodHandler struct {
	s *service.MigrationMethodService
}

func NewMigrationMethodHandler(s *service.MigrationMethodService) *MigrationMethodHandler {
	return &MigrationMethodHandler{s: s}
}

func (h *MigrationMethodHandler) GetGrid(c *gin.Context) {
	ctx := c.Request.Context()
	currentUserID := currentCreatorID()

	minTime, errMin := strconv.ParseFloat(c.Query("min_time"), 64)
	maxTime, errMax := strconv.ParseFloat(c.Query("max_time"), 64)

	var (
		methods []service.MigrationMethodView
		err     error
	)

	if errMin != nil || errMax != nil || minTime < 0 || maxTime < minTime {
		minTime, maxTime = 0.01, 1.00
		methods, err = h.s.GetPublished(ctx, currentUserID)
	} else {
		methods, err = h.s.GetPublishedByTime(ctx, minTime, maxTime, currentUserID)
	}

	if err != nil {
		methods = []service.MigrationMethodView{}
	}

	page := GridPageData{
		Methods: methods,
		MinTime: minTime,
		MaxTime: maxTime,
	}

	c.JSON(http.StatusOK, page)
}

func (h *MigrationMethodHandler) GetFeedItem(c *gin.Context) {
	idStr := c.Param("id")
	ctx := c.Request.Context()
	currentUserID := currentCreatorID()

	var (
		method service.MigrationMethodView
		err    error
	)

	if idStr == "" {
		var published []service.MigrationMethodView
		published, err = h.s.GetPublished(ctx, currentUserID)
		if err != nil || len(published) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Услуга не найдена"})
			return
		}
		method = published[0]
	} else {
		var id int64
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
			return
		}

		if c.Query("next") == "true" {
			method, err = h.s.GetNextPublishedAfterID(ctx, id, currentUserID)
		} else {
			method, err = h.s.GetByID(ctx, id, currentUserID)
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Услуга не найдена"})
			return
		}
	}

	c.JSON(http.StatusOK, FeedPageData{Method: method})
}

func (h *MigrationMethodHandler) ShowAddMethodPage(c *gin.Context) {
	draft, exists, err := h.s.GetDraft(c.Request.Context(), currentCreatorID())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить черновик"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Черновик не найден"})
		return
	}

	page := AddPageData{
		Draft:  draft,
		Exists: exists,
	}
	c.JSON(http.StatusOK, page)
}

func (h *MigrationMethodHandler) CreateDraftMethod(c *gin.Context) {
	userID := currentCreatorID()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxUploadBytes)
	form, err := c.MultipartForm()
	if err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "размер запроса превышен"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "ожидается multipart/form-data с title, image и video"})
		return
	}
	defer form.RemoveAll()

	if len(form.Value) != 1 || len(form.Value["title"]) != 1 || len(form.File) != 2 || len(form.File["image"]) != 1 || len(form.File["video"]) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нужны только title, image и video"})
		return
	}
	imageHeader := form.File["image"][0]
	videoHeader := form.File["video"][0]
	if imageHeader.Size <= 0 || videoHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "изображение и видео обязательны"})
		return
	}
	if imageHeader.Size > service.MaxImageBytes || videoHeader.Size > service.MaxVideoBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "размер изображения или видео превышен"})
		return
	}
	imageFile, err := imageHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать изображение"})
		return
	}
	defer imageFile.Close()
	videoFile, err := videoHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать видео"})
		return
	}
	defer videoFile.Close()

	err = h.s.CreateDraftMethod(c.Request.Context(), form.Value["title"][0], userID,
		service.MediaInput{Reader: imageFile, Size: imageHeader.Size},
		service.MediaInput{Reader: videoFile, Size: videoHeader.Size})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrDraftTitleRequired), errors.Is(err, service.ErrDraftTitleTooLong),
			errors.Is(err, service.ErrDraftAlreadyExists), errors.Is(err, service.ErrInvalidMedia):
			status = http.StatusBadRequest
		}
		if status == http.StatusInternalServerError {
			c.JSON(status, gin.H{"error": "Не удалось создать черновик"})
			return
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Черновик создан"})
}

func (h *MigrationMethodHandler) PublishDraftMethod(c *gin.Context) {
	userID := currentCreatorID()
	var request struct {
		Description string  `json:"description"`
		TimeInGb    float64 `json:"time_in_gb"`
		Reliability float64 `json:"reliability"`
	}
	if err := bindJSON(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.s.PublishDraft(c.Request.Context(), userID, request.Description, request.TimeInGb, request.Reliability)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Черновик опубликован"})
}

func (h *MigrationMethodHandler) SoftDeleteMethod(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	err = h.s.DeleteMethod(c.Request.Context(), id, currentCreatorID())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Услуга удалена"})
}

func (h *MigrationMethodHandler) SetLike(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	var request struct {
		Like *int `json:"like"`
	}
	if err := bindJSON(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.Like == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "поле like обязательно"})
		return
	}

	err = h.s.SetLike(c.Request.Context(), id, currentCreatorID(), *request.Like)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrInvalidLike):
			status = http.StatusBadRequest
		case errors.Is(err, ds.ErrMigrationMethodNotFound):
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"like": *request.Like})
}
