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

	minTime, errMin := strconv.ParseFloat(c.Query("min_time"), 64)
	maxTime, errMax := strconv.ParseFloat(c.Query("max_time"), 64)

	var (
		methods []service.MigrationMethodView
		err     error
	)

	if errMin != nil || errMax != nil || minTime < 0 || maxTime < minTime {
		minTime, maxTime = 0.01, 1.00
		methods, err = h.s.GetPublished(ctx)
	} else {
		methods, err = h.s.GetPublishedByTime(ctx, minTime, maxTime)
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
	h.getFeedItem(c, false)
}

func (h *MigrationMethodHandler) GetNextFeedItem(c *gin.Context) {
	h.getFeedItem(c, true)
}

func (h *MigrationMethodHandler) getFeedItem(c *gin.Context, next bool) {
	idStr := c.Param("id")
	ctx := c.Request.Context()

	var (
		method service.MigrationMethodView
		err    error
	)

	if idStr == "" {
		var published []service.MigrationMethodView
		published, err = h.s.GetPublished(ctx)
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

		if next {
			method, err = h.s.GetNextPublishedAfterID(ctx, id)
		} else {
			method, err = h.s.GetByID(ctx, id)
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
	var request struct {
		Title string `json:"title"`
	}
	if err := bindJSON(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.s.CreateDraftMethod(c.Request.Context(), request.Title, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
