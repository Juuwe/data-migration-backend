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
		"GET /api/methods/feed/:id":  {},
		"GET /api/methods/draft":     {},
		"POST /api/methods":          {},
		"PUT /api/methods/publish":   {},
		"DELETE /api/methods/:id":    {},
		"POST /api/methods/:id/like": {},
		"POST /api/users":            {},
		"POST /api/users/login":      {},
		"POST /api/users/logout":     {},
	}

	for _, route := range router.Routes() {
		if route.Path != "/static/*filepath" && (len(route.Path) < len("/api/") || route.Path[:len("/api/")] != "/api/") {
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
