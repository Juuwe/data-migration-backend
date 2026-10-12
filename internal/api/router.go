package api

import (
	"context"
	"errors"
	"os"

	"github.com/Juuwe/data-migration-backend/internal/app/handler"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/Juuwe/data-migration-backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(svc *service.MigrationMethodService, userSvc *service.UserService) *gin.Engine {
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20

	staticDir := "./static"
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		staticDir = "internal/static"
	}
	r.Static("/static", staticDir)
	r.StaticFile("/docs/swagger.yaml", "./docs/swagger.yaml")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/docs/swagger.yaml")))

	migrationMethodsHandler := handler.NewMigrationMethodHandler(svc)
	userHandler := handler.NewUserHandler(userSvc)
	authenticate := userAuthenticator(userSvc)
	optionalAuth := middleware.Auth(authenticate, false)
	requiredAuth := middleware.Auth(authenticate, true)

	r.GET("/api/methods", optionalAuth, migrationMethodsHandler.GetGrid)
	r.GET("/api/methods/feed", optionalAuth, migrationMethodsHandler.GetFeedItem)
	r.GET("/api/methods/draft", requiredAuth, migrationMethodsHandler.ShowAddMethodPage)
	r.POST("/api/methods", requiredAuth, migrationMethodsHandler.CreateDraftMethod)
	r.PUT("/api/methods/publish", requiredAuth, migrationMethodsHandler.PublishDraftMethod)
	r.DELETE("/api/methods/:id", requiredAuth, migrationMethodsHandler.SoftDeleteMethod)
	r.POST("/api/methods/:id/like", requiredAuth, migrationMethodsHandler.SetLike)
	r.POST("/api/users", userHandler.Register)
	r.POST("/api/users/login", userHandler.Authenticate)
	r.POST("/api/users/refresh", userHandler.Refresh)
	r.POST("/api/users/logout", requiredAuth, userHandler.Logout)

	return r
}

func userAuthenticator(users *service.UserService) middleware.AuthenticateFunc {
	return func(ctx context.Context, token string) (context.Context, error) {
		userID, err := users.UserIDFromToken(ctx, token)
		if errors.Is(err, auth.ErrInvalidToken) {
			return nil, middleware.ErrUnauthorized
		}
		if err != nil {
			return nil, err
		}
		return auth.WithUserID(ctx, userID), nil
	}
}
