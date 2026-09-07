package handler

import (
	"net/http"
	"strconv"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	service *service.MigrationMethodService
}

type FeedPageData struct {
	Method service.MigrationMethodView
}

func NewFeedHandler(s *service.MigrationMethodService) *FeedHandler {
	return &FeedHandler{service: s}
}

func (h *FeedHandler) Feed(c *gin.Context) {
	idStr := c.Query("id")
	nextStr := c.Query("next")
	ctx := c.Request.Context() // Используем контекст запроса

	var (
		method service.MigrationMethodView
		err    error
	)

	if idStr == "" {
		var published []service.MigrationMethodView
		published, err = h.service.GetPublished(ctx)
		if err != nil || len(published) == 0 {
			c.HTML(http.StatusNotFound, "feed.html", NewPageContext("Ошибка", "feed", false, FeedPageData{}))
			return
		}
		method = published[0]
	} else {
		var id int
		id, err = strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "feed.html", NewPageContext("Ошибка ID", "feed", false, FeedPageData{}))
			return
		}

		if nextStr == "true" {
			method, err = h.service.GetNextPublishedAfterID(ctx, id)
		} else {
			method, err = h.service.GetByID(ctx, id)
		}

		if err != nil {
			c.HTML(http.StatusNotFound, "feed.html", NewPageContext("Не найдено", "feed", false, FeedPageData{}))
			return
		}
	}

	page := NewPageContext("Лента", "feed", false, FeedPageData{Method: method})
	c.HTML(http.StatusOK, "feed", page)
}
