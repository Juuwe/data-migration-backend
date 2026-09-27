package main

import (
	"log"

	"github.com/Juuwe/data-migration-backend/internal/api"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/config"
	"github.com/Juuwe/data-migration-backend/internal/repository"
	"github.com/Juuwe/data-migration-backend/internal/storage"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	repo, err := repository.New(repository.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
		Timezone: cfg.Database.Timezone,
	})
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}

	objectStore, err := storage.NewMinIO(cfg.Storage)
	if err != nil {
		log.Fatalf("Ошибка настройки MinIO: %v", err)
	}
	svc := service.NewMigrationMethodService(repo, service.NewMediaService(objectStore))
	userSvc := service.NewUserService(repo)

	r := api.NewRouter(svc, userSvc)

	log.Printf("Сервер запущен на http://localhost:%d", cfg.WebServer.Port)
	if err := r.Run(cfg.WebServer.Address()); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
