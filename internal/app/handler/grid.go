package handler

import (
	"net/http"
	"strconv"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-gonic/gin"
)

type GridHandler struct {
	service *service.MigrationMethodService
}

type GridPageData struct {
	Methods []service.MigrationMethodView
	MinTime float64
	MaxTime float64
}

func NewGridHandler(s *service.MigrationMethodService) *GridHandler {
	return &GridHandler{service: s}
}

func (h *GridHandler) Grid(c *gin.Context) {
	ctx := c.Request.Context()

	minTime, errMin := strconv.ParseFloat(c.Query("min_time"), 64)
	maxTime, errMax := strconv.ParseFloat(c.Query("max_time"), 64)

	var (
		methods []service.MigrationMethodView
		err     error
	)

	if errMin != nil || errMax != nil {
		minTime, maxTime = 0.01, 1.00
		methods, err = h.service.GetPublished(ctx)
	} else {
		methods, err = h.service.GetPublishedByTime(ctx, minTime, maxTime)
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
