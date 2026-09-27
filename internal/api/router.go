package api

import (
	"os"

	"github.com/Juuwe/data-migration-backend/internal/app/handler"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-gonic/gin"
)

func NewRouter(svc *service.MigrationMethodService, userSvc *service.UserService) *gin.Engine {
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20

	staticDir := "./static"
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		staticDir = "internal/static"
	}
	r.Static("/static", staticDir)

	migrationMethodsHandler := handler.NewMigrationMethodHandler(svc)
	userHandler := handler.NewUserHandler(userSvc)

	r.GET("/api/methods", migrationMethodsHandler.GetGrid)
	r.GET("/api/methods/feed", migrationMethodsHandler.GetFeedItem)
	r.GET("/api/methods/feed/:id", migrationMethodsHandler.GetFeedItem)
	r.GET("/api/methods/draft", migrationMethodsHandler.ShowAddMethodPage)
	r.POST("/api/methods", migrationMethodsHandler.CreateDraftMethod)
	r.PUT("/api/methods/publish", migrationMethodsHandler.PublishDraftMethod)
	r.DELETE("/api/methods/:id", migrationMethodsHandler.SoftDeleteMethod)
	r.POST("/api/methods/:id/like", migrationMethodsHandler.SetLike)
	r.POST("/api/users", userHandler.Register)
	r.POST("/api/users/login", userHandler.Authenticate)
	r.POST("/api/users/logout", userHandler.Logout)

	return r
}
