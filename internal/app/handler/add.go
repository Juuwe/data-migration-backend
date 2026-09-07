package handler

import (
	"net/http"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-gonic/gin"
)

type AddHandler struct {
	s *service.MigrationMethodService
}

type AddPageData struct {
	Draft service.MigrationMethodView
}

func NewAddHandler(s *service.MigrationMethodService) *AddHandler {
	return &AddHandler{s: s}
}

func (h *AddHandler) Add(c *gin.Context) {
	draft, err := h.s.GetDraft(c.Request.Context())
	if err != nil {
		draft = service.MigrationMethodView{}
	}

	page := NewPageContext("Новая услуга миграции", "add", true, AddPageData{Draft: draft})
	c.HTML(http.StatusOK, "add", page)
}
