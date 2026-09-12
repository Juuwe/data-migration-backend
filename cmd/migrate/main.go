package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=postgres password=postgres dbname=migration_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Чтение файла миграции
	sqlScript, err := os.ReadFile("migrations/init.sql")
	if err != nil {
		log.Fatalf("Ошибка чтения файла migrations/init.sql: %v", err)
	}

	// Выполнение SQL-запросов
	err = db.Exec(string(sqlScript)).Error
	if err != nil {
		log.Fatalf("Ошибка выполнения миграции: %v", err)
	}

	log.Println("Успешно: таблицы созданы, первичные данные загружены!")
}
