// Package migrations применяет SQL-миграции при старте приложения.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

//go:embed 002_indexes.sql
var files embed.FS

// Apply выполняет миграции из migrations/002_indexes.sql.
// Миграция идемпотентна: CREATE INDEX IF NOT EXISTS и ANALYZE.
func Apply(ctx context.Context, db *gorm.DB) error {
	raw, err := files.ReadFile("002_indexes.sql")
	if err != nil {
		return fmt.Errorf("read 002_indexes.sql: %w", err)
	}

	for _, statement := range splitStatements(string(raw)) {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("execute statement %q: %w", statement, err)
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	lines := strings.Split(sql, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		clean = append(clean, line)
	}

	var statements []string
	for _, part := range strings.Split(strings.Join(clean, "\n"), ";") {
		if statement := strings.TrimSpace(part); statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}
