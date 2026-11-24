package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/config"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/go-faker/faker/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Нужно строго больше 100000 опубликованных услуг.
const targetPublished = 100500

// Короткое английское описание для CLIP-поиска по изображению.
const (
	minDescriptionLen = 50
	maxDescriptionLen = 100
)

var (
	descriptionVerbs = []string{
		"Migrates", "Transfers", "Moves", "Copies", "Synchronizes", "Replicates",
		"Streams", "Validates", "Converts", "Archives", "Loads", "Reconciles",
	}
	descriptionObjects = []string{
		"relational databases", "large datasets", "archived records", "legacy systems",
		"cloud storage", "data warehouses", "transaction logs", "backup copies",
	}
	descriptionDetails = []string{
		"with minimal downtime", "without data loss", "in a single window",
		"using parallel streams", "with integrity checks", "under production load",
		"between cloud regions", "across data centers", "over encrypted channels",
		"with schema mapping",
	}
	descriptionScales = []string{
		"", "up to 4 TB", "up to 16 TB", "over 100 GB", "in under 2 hours", "per migration run",
	}
)

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
	refreshDescriptions := flag.Bool("refresh-descriptions", false, "перегенерировать description_en для сгенерированных строк")
	flag.Parse()

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
	if *refreshDescriptions {
		// Английские описания перегенерируются, описания сид-услуг (на русском) сохраняются.
		if err := db.WithContext(ctx).
			Exec("UPDATE migration_methods SET description_en = NULL WHERE description IS NULL OR description !~ '[А-Яа-яЁё]'").
			Error; err != nil {
			log.Fatalf("reset description_en: %v", err)
		}
	}

	backfillEnglishDescriptions(ctx, db, *refreshDescriptions)
	log.Printf("seed finished")
}

// englishDescription возвращает английское описание длиной 50–100 символов:
// существующее подходящее описание сохраняется, иначе генерируется новое.
// При force существующее описание не переиспользуется.
func englishDescription(source *string, force bool) string {
	if !force && source != nil {
		trimmed := strings.TrimSpace(*source)
		if isShortEnglish(trimmed) {
			return trimmed
		}
	}

	text := generatedDescription()
	for len(text) < minDescriptionLen {
		text = strings.TrimSpace(text + " " + faker.Word())
	}
	if len(text) > maxDescriptionLen {
		cut := text[:maxDescriptionLen]
		if index := strings.LastIndex(cut, " "); index >= minDescriptionLen {
			cut = cut[:index]
		}
		text = cut
	}
	return text
}

// generatedDescription собирает осмысленное английское описание услуги
// миграции длиной 50–100 символов из тематических шаблонов.
func generatedDescription() string {
	for attempt := 0; attempt < 100; attempt++ {
		firstDetail := pick(descriptionDetails)
		text := fmt.Sprintf("%s %s %s",
			pick(descriptionVerbs), pick(descriptionObjects), firstDetail)
		if scale := pick(descriptionScales); scale != "" {
			secondDetail := pick(descriptionDetails)
			for secondDetail == firstDetail {
				secondDetail = pick(descriptionDetails)
			}
			text = fmt.Sprintf("%s %s %s", text, scale, secondDetail)
		}
		text = strings.Join(strings.Fields(text), " ") + "."
		if len(text) >= minDescriptionLen && len(text) <= maxDescriptionLen {
			return text
		}
	}
	return "Transfers large datasets between systems with integrity checks."
}

func pick(values []string) string {
	return values[rand.Intn(len(values))]
}

func isShortEnglish(text string) bool {
	if len(text) < minDescriptionLen || len(text) > maxDescriptionLen {
		return false
	}
	for _, r := range text {
		if r < 0x20 || r > 0x7E {
			return false
		}
	}
	return true
}

// backfillEnglishDescriptions заполняет description_en для строк, созданных
// до появления колонки, чтобы поиск по изображению работал на всём каталоге.
// При force описания перегенерируются, даже если подходящий текст уже есть.
func backfillEnglishDescriptions(ctx context.Context, db *gorm.DB, force bool) {
	type row struct {
		ID          int64
		Description *string
	}

	var rows []row
	if err := db.WithContext(ctx).
		Model(&ds.MigrationMethod{}).
		Select("id", "description").
		Where("description_en IS NULL").
		Find(&rows).Error; err != nil {
		log.Fatalf("load methods without description_en: %v", err)
	}
	if len(rows) == 0 {
		return
	}

	const batchSize = 500
	for start := 0; start < len(rows); start += batchSize {
		end := min(start+batchSize, len(rows))

		placeholders := make([]string, 0, end-start)
		arguments := make([]any, 0, 2*(end-start))
		for _, item := range rows[start:end] {
			placeholders = append(placeholders, "(?::bigint, ?::varchar)")
			arguments = append(arguments, item.ID, englishDescription(item.Description, force))
		}

		query := "UPDATE migration_methods AS methods SET description_en = data.description_en " +
			"FROM (VALUES " + strings.Join(placeholders, ", ") + ") AS data(id, description_en) " +
			"WHERE methods.id = data.id"
		if err := db.WithContext(ctx).Exec(query, arguments...).Error; err != nil {
			log.Fatalf("backfill description_en: %v", err)
		}
	}
	log.Printf("description_en заполнено для %d методов", len(rows))
}

func seed(ctx context.Context, db *gorm.DB, users []ds.User, needed int, existing int64) {
	batch := make([]ds.MigrationMethod, 0, 1000)
	created := 0

	for i := 0; i < needed; i++ {
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(365*5))
		timeInGb := math.Round((0.01+rand.Float64()*0.99)*100) / 100
		reliability := math.Round((0.95+rand.Float64()*0.0499)*10000) / 10000
		description := faker.Sentence()
		descriptionEn := englishDescription(&description, true)
		item := media[rand.Intn(len(media))]
		title := fmt.Sprintf("%s %d", faker.Word(), existing+int64(i)+1)

		batch = append(batch, ds.MigrationMethod{
			CreatorID:     users[rand.Intn(len(users))].ID,
			Title:         title,
			Description:   &description,
			DescriptionEn: &descriptionEn,
			Status:        ds.StatusPublished,
			ImageKey:      item.image,
			VideoKey:      item.video,
			TimeInGb:      &timeInGb,
			Reliability:   &reliability,
			CreatedAt:     createdAt,
			PublishedAt:   &createdAt,
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
