package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

type feedRepositoryStub struct {
	service.MigrationMethodRepository
}

func (feedRepositoryStub) FindByID(context.Context, int64) (ds.MigrationMethod, error) {
	return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
}

func TestGetFeedItemNotFoundReturnsEmpty404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := service.NewMigrationMethodService(feedRepositoryStub{})
	router := gin.New()
	router.GET("/feed", NewMigrationMethodHandler(svc).GetFeedItem)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/feed?id=999", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusNotFound)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("response body = %q, want an empty body", response.Body.String())
	}
}
