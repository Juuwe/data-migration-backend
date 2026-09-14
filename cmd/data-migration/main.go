package main

import (
	"log"

	"github.com/Juuwe/data-migration-backend/internal/api"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/config"
	"github.com/Juuwe/data-migration-backend/internal/repository"
	miniostorage "github.com/Juuwe/data-migration-backend/internal/storage/minio"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Fatalf("Ошибка настройки MinIO: %v", err)
	}

	storage := miniostorage.New(minioClient, cfg.MinIO.Bucket, cfg.MinIO.BaseURL)
	svc := service.NewMigrationMethodService(repo, storage)

	r := api.NewRouter(svc)

	log.Printf("Сервер запущен на http://localhost:%d", cfg.WebServer.Port)
	if err := r.Run(cfg.WebServer.Address()); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
