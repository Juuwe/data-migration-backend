// Package migrations применяет SQL-миграции при старте приложения.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

//go:embed *.sql
var files embed.FS

// Apply выполняет все миграции из каталога migrations, кроме init.sql
// (init.sql применяется отдельно при создании схемы). Миграции идемпотентны.
func Apply(ctx context.Context, db *gorm.DB) error {
	entries, err := files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") || name == "init.sql" {
			continue
		}

		raw, err := files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		for _, statement := range splitStatements(string(raw)) {
			if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
				return fmt.Errorf("execute %s statement %q: %w", name, statement, err)
			}
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
