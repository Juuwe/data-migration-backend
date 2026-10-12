// @title Data Migration API
// @version 1.0
// @description API методов миграции данных. Для защищенных операций нужен JWT в заголовке Authorization: Bearer <token>. Выход отзывает предъявленные access- и refresh-токены. Ошибки возвращаются без тела ответа.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Juuwe/data-migration-backend/internal/api"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/Juuwe/data-migration-backend/internal/config"
	"github.com/Juuwe/data-migration-backend/internal/repository"
	"github.com/Juuwe/data-migration-backend/internal/storage"
	"github.com/Juuwe/data-migration-backend/internal/tokenstore"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load(".env")
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}
	tokens, err := auth.NewManager(cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("ошибка настройки JWT: %w", err)
	}

	repo, err := repository.New(cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("ошибка подключения к PostgreSQL: %w", err)
	}
	defer repo.Close()
	tokenStore, err := tokenstore.New(context.Background(), cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		return fmt.Errorf("ошибка подключения к Redis: %w", err)
	}
	defer tokenStore.Close()

	objectStore, err := storage.NewMinIO(cfg.Storage)
	if err != nil {
		return fmt.Errorf("ошибка настройки MinIO: %w", err)
	}
	svc := service.NewMigrationMethodService(repo, service.NewMediaService(objectStore))
	userSvc := service.NewUserService(repo, tokens, tokenStore)

	r := api.NewRouter(svc, userSvc)

	log.Printf("Сервер запущен на http://localhost:%d", cfg.WebServer.Port)
	if err := r.Run(cfg.WebServer.Address()); err != nil {
		return fmt.Errorf("ошибка запуска сервера: %w", err)
	}
	return nil
}
