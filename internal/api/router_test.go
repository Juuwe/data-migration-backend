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
		"GET /api/v1/methods":           {},
		"GET /api/v1/methods/feed":      {},
		"GET /api/v1/methods/:id":       {},
		"GET /api/v1/methods/:id/next":  {},
		"GET /api/v1/methods/draft":     {},
		"POST /api/v1/methods":          {},
		"PUT /api/v1/methods/publish":   {},
		"DELETE /api/v1/methods/:id":    {},
		"POST /api/v1/methods/:id/like": {},
		"POST /api/v1/users":            {},
		"POST /api/v1/users/login":      {},
		"POST /api/v1/users/logout":     {},
	}

	for _, route := range router.Routes() {
		if route.Path != "/static/*filepath" && (len(route.Path) < len("/api/v1/") || route.Path[:len("/api/v1/")] != "/api/v1/") {
			t.Fatalf("route %s %s has an unexpected path", route.Method, route.Path)
		}
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) != 0 {
		t.Fatalf("router is missing routes: %v", want)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/static/images/default.svg", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /static/images/default.svg: status = %d, want 200", response.Code)
	}
}
