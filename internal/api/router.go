package api

import (
	"os"

	"github.com/Juuwe/data-migration-backend/internal/app/handler"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-gonic/gin"
)

func NewRouter(svc *service.MigrationMethodService, userSvc *service.UserService) *gin.Engine {
	r := gin.Default()

	// Находим папку static в корне или в internal/static
	staticDir := "./static"
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		staticDir = "internal/static"
	}
	r.Static("/static", staticDir)

	migrationMethodsHandler := handler.NewMigrationMethodHandler(svc)
	userHandler := handler.NewUserHandler(userSvc)

	r.GET("/api/v1/methods", migrationMethodsHandler.GetGrid)
	r.GET("/api/v1/methods/feed", migrationMethodsHandler.GetFeedItem)
	r.GET("/api/v1/methods/:id", migrationMethodsHandler.GetFeedItem)
	r.GET("/api/v1/methods/:id/next", migrationMethodsHandler.GetNextFeedItem)
	r.GET("/api/v1/methods/draft", migrationMethodsHandler.ShowAddMethodPage)
	r.POST("/api/v1/methods", migrationMethodsHandler.CreateDraftMethod)
	r.PUT("/api/v1/methods/publish", migrationMethodsHandler.PublishDraftMethod)
	r.DELETE("/api/v1/methods/:id", migrationMethodsHandler.SoftDeleteMethod)
	r.POST("/api/v1/methods/:id/like", migrationMethodsHandler.SetLike)
	r.POST("/api/v1/users", userHandler.Register)
	r.POST("/api/v1/users/login", userHandler.Authenticate)
	r.POST("/api/v1/users/logout", userHandler.Logout)

	return r
}
