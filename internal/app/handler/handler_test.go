package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetFeedItemNotFoundReturnsJSON404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := service.NewMigrationMethodService(feedRepositoryStub{})
	router := gin.New()
	router.GET("/api/v1/methods/:id", NewMigrationMethodHandler(svc).GetFeedItem)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/methods/999", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusNotFound)
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", response.Header().Get("Content-Type"))
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Fatalf("response body = %q, want a JSON error: %v", response.Body.String(), err)
	}
}

type jsonRepositoryStub struct {
	service.MigrationMethodRepository
	draft     ds.MigrationMethod
	published []ds.MigrationMethod
	created   *ds.MigrationMethod
	updated   *ds.MigrationMethod
	deletedID int64
	nextID    int64
	likeID    int64
	likeUser  int64
	likeValue int
	likeCalls int
}

func (r *jsonRepositoryStub) FindPublished(context.Context) ([]ds.MigrationMethod, error) {
	return r.published, nil
}

func (r *jsonRepositoryStub) FindByID(_ context.Context, id int64) (ds.MigrationMethod, error) {
	for _, method := range r.published {
		if method.ID == id {
			return method, nil
		}
	}
	return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
}

func (r *jsonRepositoryStub) FindNextPublishedAfterID(_ context.Context, id int64) (ds.MigrationMethod, error) {
	r.nextID = id
	for _, method := range r.published {
		if method.ID > id {
			return method, nil
		}
	}
	return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
}

func (r *jsonRepositoryStub) CountMethodLikesByID(context.Context, int64) (int, error) {
	return 0, nil
}

func (r *jsonRepositoryStub) CountLikesByMethodIDs(context.Context, []int64) (map[int64]int, error) {
	return map[int64]int{}, nil
}

func (r *jsonRepositoryStub) FindDraft(context.Context, int64) (ds.MigrationMethod, error) {
	if r.draft.ID == 0 {
		return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
	}
	return r.draft, nil
}

func (r *jsonRepositoryStub) Create(_ context.Context, method *ds.MigrationMethod) error {
	created := *method
	created.ID = 7
	r.created = &created
	r.draft = created
	return nil
}

func (r *jsonRepositoryStub) Update(_ context.Context, method *ds.MigrationMethod) error {
	updated := *method
	r.updated = &updated
	return nil
}

func (r *jsonRepositoryStub) SoftDeleteSQL(_ context.Context, id, creatorID int64) error {
	r.deletedID = id
	if creatorID != currentCreatorID() {
		return errors.New("unexpected creator ID")
	}
	return nil
}

func (r *jsonRepositoryStub) SetLike(_ context.Context, methodID, userID int64, like int) error {
	r.likeID = methodID
	r.likeUser = userID
	r.likeValue = like
	r.likeCalls++
	return nil
}

func TestDraftWorkflowReturnsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jsonRepositoryStub{}
	handler := NewMigrationMethodHandler(service.NewMigrationMethodService(repo))
	router := gin.New()
	router.GET("/api/v1/methods/draft", handler.ShowAddMethodPage)
	router.POST("/api/v1/methods", handler.CreateDraftMethod)
	router.PUT("/api/v1/methods/publish", handler.PublishDraftMethod)
	router.DELETE("/api/v1/methods/:id", handler.SoftDeleteMethod)

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodGet, "/api/v1/methods/draft", "", http.StatusNotFound},
		{http.MethodPost, "/api/v1/methods", `{"title":"Новая услуга"}`, http.StatusCreated},
		{http.MethodGet, "/api/v1/methods/draft", "", http.StatusOK},
		{http.MethodPut, "/api/v1/methods/publish", `{"description":"Описание","time_in_gb":0.25,"reliability":0.9}`, http.StatusOK},
		{http.MethodDelete, "/api/v1/methods/7", "", http.StatusOK},
	}

	for _, test := range tests {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		if test.body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		router.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("%s %s: status = %d, want %d; body = %q", test.method, test.path, response.Code, test.status, response.Body.String())
		}
		if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") || !json.Valid(response.Body.Bytes()) {
			t.Fatalf("%s %s: response is not JSON: %q", test.method, test.path, response.Body.String())
		}
		if test.method == http.MethodGet && test.status == http.StatusNotFound {
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] == nil || body["draft"] != nil {
				t.Fatalf("missing draft response = %q", response.Body.String())
			}
		}
	}

	if repo.created == nil || repo.created.Title != "Новая услуга" {
		t.Fatalf("created draft = %+v", repo.created)
	}
	if repo.updated == nil || repo.updated.Status != ds.StatusPublished {
		t.Fatalf("published draft = %+v", repo.updated)
	}
	if repo.deletedID != 7 {
		t.Fatalf("deleted ID = %d, want 7", repo.deletedID)
	}
}

func TestMethodMutationsRejectSystemFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jsonRepositoryStub{}
	handler := NewMigrationMethodHandler(service.NewMigrationMethodService(repo))
	router := gin.New()
	router.POST("/api/v1/methods", handler.CreateDraftMethod)
	router.PUT("/api/v1/methods/publish", handler.PublishDraftMethod)

	for _, test := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/methods", `{"title":"Услуга","status":"published"}`},
		{http.MethodPost, "/api/v1/methods", `{"title":"Услуга","image_url":"/photo.png","video_url":"/video.mp4"}`},
		{http.MethodPut, "/api/v1/methods/publish", `{"description":"Описание","time_in_gb":0.25,"reliability":0.9,"creator_id":77}`},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !json.Valid(response.Body.Bytes()) {
			t.Fatalf("%s %s: status = %d, body = %q", test.method, test.path, response.Code, response.Body.String())
		}
	}
	if repo.created != nil || repo.updated != nil {
		t.Fatalf("repository was called for system fields: created = %+v, updated = %+v", repo.created, repo.updated)
	}
}

func TestMethodRoutesReadIDsFromPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jsonRepositoryStub{published: []ds.MigrationMethod{
		{ID: 3, Title: "Первая", Status: ds.StatusPublished},
		{ID: 7, Title: "Вторая", Status: ds.StatusPublished},
	}}
	handler := NewMigrationMethodHandler(service.NewMigrationMethodService(repo))
	router := gin.New()
	router.GET("/api/v1/methods", handler.GetGrid)
	router.GET("/api/v1/methods/feed", handler.GetFeedItem)
	router.GET("/api/v1/methods/:id", handler.GetFeedItem)
	router.GET("/api/v1/methods/:id/next", handler.GetNextFeedItem)

	tests := []struct {
		path   string
		field  string
		wantID float64
	}{
		{"/api/v1/methods", "methods", 3},
		{"/api/v1/methods/feed", "method", 3},
		{"/api/v1/methods/7", "method", 7},
		{"/api/v1/methods/3/next", "method", 7},
	}
	for _, test := range tests {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s: status = %d, body = %q", test.path, response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("GET %s: invalid JSON: %v", test.path, err)
		}
		var method map[string]any
		if test.field == "methods" {
			method = body[test.field].([]any)[0].(map[string]any)
		} else {
			method = body[test.field].(map[string]any)
		}
		if method["id"] != test.wantID {
			t.Fatalf("GET %s: id = %v, want %v", test.path, method["id"], test.wantID)
		}
		if _, exists := method["creator"]; exists {
			t.Fatalf("GET %s: creator was exposed in JSON", test.path)
		}
	}
	if repo.nextID != 3 {
		t.Fatalf("next lookup ID = %d, want 3", repo.nextID)
	}
}

func TestSetLikeUsesCurrentUserAndAcceptsOnlyZeroOrOne(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jsonRepositoryStub{published: []ds.MigrationMethod{{ID: 7, Status: ds.StatusPublished}}}
	router := gin.New()
	router.POST("/api/v1/methods/:id/like", NewMigrationMethodHandler(service.NewMigrationMethodService(repo)).SetLike)

	for _, test := range []struct {
		path   string
		body   string
		status int
		value  int
	}{
		{"/api/v1/methods/7/like", `{"like":1}`, http.StatusOK, 1},
		{"/api/v1/methods/7/like", `{"like":0}`, http.StatusOK, 0},
		{"/api/v1/methods/7/like", `{"like":2}`, http.StatusBadRequest, 0},
		{"/api/v1/methods/7/like", `{"like":1,"status":"published"}`, http.StatusBadRequest, 0},
		{"/api/v1/methods/7/like", `{}`, http.StatusBadRequest, 0},
		{"/api/v1/methods/bad/like", `{"like":1}`, http.StatusBadRequest, 0},
		{"/api/v1/methods/8/like", `{"like":1}`, http.StatusNotFound, 0},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		router.ServeHTTP(response, request)
		if response.Code != test.status || !json.Valid(response.Body.Bytes()) {
			t.Fatalf("POST %s %s: status = %d, body = %q", test.path, test.body, response.Code, response.Body.String())
		}
		if test.status == http.StatusOK {
			var body map[string]int
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["like"] != test.value {
				t.Fatalf("POST %s %s: response = %q", test.path, test.body, response.Body.String())
			}
			if repo.likeID != 7 || repo.likeUser != currentCreatorID() || repo.likeValue != test.value {
				t.Fatalf("like delegation = method %d, user %d, value %d", repo.likeID, repo.likeUser, repo.likeValue)
			}
		}
	}
	if repo.likeCalls != 2 {
		t.Fatalf("SetLike repository calls = %d, want 2", repo.likeCalls)
	}
}
