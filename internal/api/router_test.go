package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/gin-gonic/gin"
)

func TestRouterContainsAssignmentRoutes(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Clean(filepath.Join(workingDirectory, "../.."))
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	gin.SetMode(gin.TestMode)
	router := NewRouter(service.NewMigrationMethodService(nil, nil))

	want := map[string]struct{}{
		"GET /feed":                {},
		"GET /grid":                {},
		"GET /add":                 {},
		"POST /add":                {},
		"POST /add/publish":        {},
		"POST /methods/:id/delete": {},
	}

	for _, route := range router.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) != 0 {
		t.Fatalf("router is missing routes: %v", want)
	}
}
