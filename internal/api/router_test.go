package api

import (
	"net/http"
	"net/http/httptest"
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
	router := NewRouter(service.NewMigrationMethodService(nil), service.NewUserService(nil))

	want := map[string]struct{}{
		"GET /api/methods":           {},
		"GET /api/methods/feed":      {},
		"GET /api/methods/draft":     {},
		"POST /api/methods":          {},
		"PUT /api/methods/publish":   {},
		"DELETE /api/methods/:id":    {},
		"POST /api/methods/:id/like": {},
		"POST /api/users":            {},
		"POST /api/users/login":      {},
		"POST /api/users/logout":     {},
	}

	apiRoutes, getRoutes := 0, 0
	for _, route := range router.Routes() {
		if route.Path == "/static/*filepath" {
			continue
		}
		key := route.Method + " " + route.Path
		if _, exists := want[key]; !exists {
			t.Fatalf("unexpected API route: %s", key)
		}
		apiRoutes++
		if route.Method == http.MethodGet {
			getRoutes++
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("router is missing routes: %v", want)
	}
	if apiRoutes != 10 || getRoutes != 3 {
		t.Fatalf("API routes: %d total, %d GET; want 10 total, 3 GET", apiRoutes, getRoutes)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/static/images/default.svg", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /static/images/default.svg: status = %d, want 200", response.Code)
	}
}
