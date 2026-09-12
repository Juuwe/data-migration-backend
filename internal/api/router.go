package api

import (
	"os"

	"github.com/Juuwe/data-migration-backend/internal/app/handler"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

func createMyRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	r.AddFromFiles("feed",
		"internal/templates/base.html",
		"internal/templates/bottom_nav.html",
		"internal/templates/service_card.html",
		"internal/templates/feed.html",
	)
	r.AddFromFiles("grid",
		"internal/templates/base.html",
		"internal/templates/bottom_nav.html",
		"internal/templates/service_card.html",
		"internal/templates/grid.html",
	)
	r.AddFromFiles("add",
		"internal/templates/base.html",
		"internal/templates/bottom_nav.html",
		"internal/templates/add.html",
	)

	return r
}

func NewRouter(svc *service.MigrationMethodService) *gin.Engine {
	r := gin.Default()

	r.HTMLRender = createMyRender()

	// Находим папку static в корне или в internal/static
	staticDir := "./static"
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		staticDir = "internal/static"
	}
	r.Static("/static", staticDir)

	migrationMethodsHandler := handler.NewMigrationMethodHandler(svc)

	r.GET("/", migrationMethodsHandler.GetFeedItem)
	r.GET("/feed", migrationMethodsHandler.GetFeedItem)
	r.GET("/grid", migrationMethodsHandler.GetGrid)
	r.GET("/add", migrationMethodsHandler.ShowAddMethodPage)

	r.POST("/add", )

	return r
}
