package main

import (
	"log"
	"os"

	"github.com/Juuwe/data-migration-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	sqlScript, err := os.ReadFile("migrations/init.sql")
	if err != nil {
		log.Fatalf("Ошибка чтения файла migrations/init.sql: %v", err)
	}

	err = db.Exec(string(sqlScript)).Error
	if err != nil {
		log.Fatalf("Ошибка выполнения миграции: %v", err)
	}

	log.Println("Успешно: таблицы созданы, первичные данные загружены!")
}
