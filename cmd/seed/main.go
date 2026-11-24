package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/config"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/go-faker/faker/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Нужно строго больше 100000 опубликованных услуг.
const targetPublished = 100500

var media = []struct {
	image string
	video string
}{
	{"online.png", "online.mp4"},
	{"offline.png", "offline.mp4"},
	{"replication.png", "replication.mp4"},
	{"hybrid.png", "hybrid.mp4"},
	{"elt.png", "elt.mp4"},
	{"physical.png", "physical.mp4"},
	{"audit.png", "audit.mp4"},
}

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	// Для массовой вставки часовой пояс не важен; UTC исключает проблемы
	// с URL-кодированием имени зоны в DSN.
	cfg.Database.Timezone = "UTC"

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	ctx := context.Background()

	var users []ds.User
	if err := db.WithContext(ctx).Find(&users).Error; err != nil {
		log.Fatalf("load users: %v", err)
	}
	if len(users) == 0 {
		log.Fatal("no users found; apply migrations/init.sql first")
	}

	var published int64
	if err := db.WithContext(ctx).
		Model(&ds.MigrationMethod{}).
		Where("status = ?", ds.StatusPublished).
		Count(&published).Error; err != nil {
		log.Fatalf("count published methods: %v", err)
	}

	needed := targetPublished - int(published)
	if needed <= 0 {
		log.Printf("already have %d published methods, nothing to seed", published)
	} else {
		seed(ctx, db, users, needed, published)
	}

	if err := db.WithContext(ctx).Exec("ANALYZE migration_methods").Error; err != nil {
		log.Fatalf("analyze migration_methods: %v", err)
	}
	log.Printf("seed finished")
}

func seed(ctx context.Context, db *gorm.DB, users []ds.User, needed int, existing int64) {
	batch := make([]ds.MigrationMethod, 0, 1000)
	created := 0

	for i := 0; i < needed; i++ {
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(365*5))
		timeInGb := math.Round((0.01+rand.Float64()*0.99)*100) / 100
		reliability := math.Round((0.95+rand.Float64()*0.0499)*10000) / 10000
		description := faker.Sentence()
		item := media[rand.Intn(len(media))]
		title := fmt.Sprintf("%s %d", faker.Word(), existing+int64(i)+1)

		batch = append(batch, ds.MigrationMethod{
			CreatorID:   users[rand.Intn(len(users))].ID,
			Title:       title,
			Description: &description,
			Status:      ds.StatusPublished,
			ImageKey:    item.image,
			VideoKey:    item.video,
			TimeInGb:    &timeInGb,
			Reliability: &reliability,
			CreatedAt:   createdAt,
			PublishedAt: &createdAt,
		})

		if len(batch) == cap(batch) {
			if err := insertBatch(ctx, db, batch); err != nil {
				log.Fatalf("insert batch after %d rows: %v", created, err)
			}
			created += len(batch)
			batch = batch[:0]
			if created%10000 == 0 {
				log.Printf("seeded %d / %d methods", created, needed)
			}
		}
	}

	if len(batch) > 0 {
		if err := insertBatch(ctx, db, batch); err != nil {
			log.Fatalf("insert final batch: %v", err)
		}
	}
}

func insertBatch(ctx context.Context, db *gorm.DB, batch []ds.MigrationMethod) error {
	return db.WithContext(ctx).CreateInBatches(batch, len(batch)).Error
}
