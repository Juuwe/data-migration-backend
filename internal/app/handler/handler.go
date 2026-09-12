package handler

import (
	"net/http"
	"strconv"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-gonic/gin"
)

type PageContext[T any] struct {
	Title        string
	ActiveNav    string
	IsLightTheme bool
	Data         T
}

func NewPageContext[T any](title, activeNav string, isLightTheme bool, data T) PageContext[T] {
	return PageContext[T]{
		Title:        title,
		ActiveNav:    activeNav,
		IsLightTheme: isLightTheme,
		Data:         data,
	}
}

type GridPageData struct {
	Methods []service.MigrationMethodView
	MinTime float64
	MaxTime float64
}

type FeedPageData struct {
	Method service.MigrationMethodView
}

type AddPageData struct {
	Draft  service.MigrationMethodView
	Exists bool
	Error  string
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

	if errMin != nil || errMax != nil {
		minTime, maxTime = 0.01, 1.00
		methods, err = h.s.GetPublished(ctx)
	} else {
		methods, err = h.s.GetPublishedByTime(ctx, minTime, maxTime)
	}

	if err != nil {
		methods = []service.MigrationMethodView{}
	}

	page := NewPageContext("Плитка услуг", "grid", true, GridPageData{
		Methods: methods,
		MinTime: minTime,
		MaxTime: maxTime,
	})

	c.HTML(http.StatusOK, "grid", page)
}

func (h *MigrationMethodHandler) GetFeedItem(c *gin.Context) {
	idStr := c.Query("id")
	nextStr := c.Query("next")
	ctx := c.Request.Context()

	var (
		method service.MigrationMethodView
		err    error
	)

	if idStr == "" {
		var published []service.MigrationMethodView
		published, err = h.s.GetPublished(ctx)
		if err != nil || len(published) == 0 {
			c.HTML(http.StatusNotFound, "feed", NewPageContext("Ошибка", "feed", false, FeedPageData{}))
			return
		}
		method = published[0]
	} else {
		var id int
		id, err = strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "feed", NewPageContext("Ошибка ID", "feed", false, FeedPageData{}))
			return
		}

		if nextStr == "true" {
			method, err = h.s.GetNextPublishedAfterID(ctx, id)
		} else {
			method, err = h.s.GetByID(ctx, id)
		}

		if err != nil {
			c.HTML(http.StatusNotFound, "feed", NewPageContext("Не найдено", "feed", false, FeedPageData{}))
			return
		}
	}

	page := NewPageContext("Лента", "feed", false, FeedPageData{Method: method})
	c.HTML(http.StatusOK, "feed", page)
}

func (h *MigrationMethodHandler) ShowAddMethodPage(c *gin.Context) {
	draft, exists, _ := h.s.GetDraft(c.Request.Context(), 1)

	page := NewPageContext("Новая услуга миграции", "add", true, AddPageData{
		Draft:  draft,
		Exists: exists,
	})
	c.HTML(http.StatusOK, "add", page)
}

func (h *MigrationMethodHandler) CreateDraftMethod(c *gin.Context) {
	userID := int64(1)
	title := c.PostForm("title")

	err := h.s.CreateDraftMethod(c.Request.Context(), title, userID)
	if err != nil {
		page := NewPageContext("Новая услуга миграции", "add", true, AddPageData{
			Error: err.Error(),
		})
		c.HTML(http.StatusBadRequest, "add", page)
		return
	}

	c.Redirect(http.StatusSeeOther, "/add")
}

func (h *MigrationMethodHandler) PublishDraftMethod(c *gin.Context) {
	userID := int64(1)
	desc := c.PostForm("description")
	timeInGb, _ := strconv.ParseFloat(c.PostForm("time_in_gb"), 64)
	reliability, _ := strconv.ParseFloat(c.PostForm("reliability"), 64)

	err := h.s.PublishDraft(c.Request.Context(), userID, desc, timeInGb, reliability)
	if err != nil {
		draft, _, _ := h.s.GetDraft(c.Request.Context(), userID)
		page := NewPageContext("Новая услуга миграции", "add", true, AddPageData{
			Draft:  draft,
			Exists: true,
			Error:  err.Error(),
		})
		c.HTML(http.StatusBadRequest, "add", page)
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func (h *MigrationMethodHandler) SoftDeleteMethod(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Некорректный ID")
		return
	}

	err = h.s.DeleteMethod(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}
