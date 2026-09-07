package main

import (
	"log"

	"github.com/Juuwe/data-migration-backend/internal/api"
	"github.com/Juuwe/data-migration-backend/internal/app/repository"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	minioClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("admin", "password123", ""),
		Secure: false,
	})
	if err != nil {
		log.Printf("Предупреждение: Не удалось подключиться к MinIO: %v", err)
	}

	storageSvc := service.NewStorageService(minioClient, "data-migration-service", "http://localhost:9000")
	repo := repository.NewInMemoryMigrationRepository()
	svc := service.NewMigrationMethodService(repo, storageSvc)

	r := api.NewRouter(svc)

	log.Println("Сервер запущен на http://localhost:8080")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
