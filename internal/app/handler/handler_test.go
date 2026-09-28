package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
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

func TestGetFeedItemNotFoundReturnsEmpty404(t *testing.T) {
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
	if response.Body.Len() != 0 {
		t.Fatalf("response body = %q, want empty", response.Body.String())
	}
}

type jsonRepositoryStub struct {
	service.MigrationMethodRepository
	draft      ds.MigrationMethod
	published  []ds.MigrationMethod
	created    *ds.MigrationMethod
	updated    *ds.MigrationMethod
	deletedID  int64
	nextID     int64
	likeID     int64
	likeUser   int64
	likeValue  int
	likeCalls  int
	likesCount int
	isLiked    bool
}

type objectStoreStub struct{}

func (objectStoreStub) Put(context.Context, string, io.ReadSeeker, int64, string) error { return nil }
func (objectStoreStub) Delete(context.Context, string) error                            { return nil }
func (objectStoreStub) PublicURL(key string) string {
	return "http://localhost:9000/data-migration-service/" + key
}

func multipartDraftRequest(t *testing.T) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Новая услуга"); err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct {
		name string
		data []byte
	}{
		{"image", []byte("\x89PNG\r\n\x1a\n")},
		{"video", []byte("\x1a\x45\xdf\xa3")},
	} {
		part, err := writer.CreateFormFile(file.name, file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/methods", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
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
	return r.likesCount, nil
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
	method.ID = 7
	created := *method
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
	if like == 1 && !r.isLiked {
		r.likesCount++
	} else if like == 0 && r.isLiked {
		r.likesCount--
	}
	r.isLiked = like == 1
	return nil
}

func TestDraftWorkflowUsesStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jsonRepositoryStub{}
	handler := NewMigrationMethodHandler(service.NewMigrationMethodService(repo, service.NewMediaService(objectStoreStub{})))
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
		{http.MethodPut, "/api/v1/methods/publish", `{"description":"Описание","time_in_gb":0.25,"reliability":0.9}`, http.StatusNotFound},
		{http.MethodPost, "/api/v1/methods", "", http.StatusCreated},
		{http.MethodGet, "/api/v1/methods/draft", "", http.StatusOK},
		{http.MethodPut, "/api/v1/methods/publish", `{"description":"Описание","time_in_gb":0.25,"reliability":0.9}`, http.StatusOK},
		{http.MethodDelete, "/api/v1/methods/7", "", http.StatusNoContent},
	}

	var createdBody []byte
	for _, test := range tests {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		if test.method == http.MethodPost {
			request = multipartDraftRequest(t)
		}
		if test.body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		router.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("%s %s: status = %d, want %d; body = %q", test.method, test.path, response.Code, test.status, response.Body.String())
		}
		if test.method == http.MethodPost && test.status == http.StatusCreated {
			if response.Header().Get("Location") != "/api/v1/methods/draft" || !json.Valid(response.Body.Bytes()) {
				t.Fatalf("POST draft: Location = %q, body = %q", response.Header().Get("Location"), response.Body.String())
			}
			var body map[string]map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["draft"]["id"] != float64(7) {
				t.Fatalf("POST draft: created object = %q, error = %v", response.Body.String(), err)
			}
			createdBody = append([]byte(nil), response.Body.Bytes()...)
		} else if test.method == http.MethodGet && test.status == http.StatusOK {
			if !json.Valid(response.Body.Bytes()) {
				t.Fatalf("GET draft: invalid JSON: %q", response.Body.String())
			}
			if !bytes.Equal(response.Body.Bytes(), createdBody) {
				t.Fatalf("GET draft = %q, POST draft = %q", response.Body.String(), createdBody)
			}
		} else if test.method == http.MethodPut && test.status == http.StatusOK {
			var body map[string]map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("PUT publish: invalid JSON: %q", response.Body.String())
			}
			method := body["method"]
			if len(method) != 9 || method["id"] != float64(7) || method["title"] != "Новая услуга" ||
				method["description"] != "Описание" || method["time_in_gb"] != 0.25 || method["reliability"] != 0.9 {
				t.Fatalf("PUT publish: method = %v", method)
			}
		} else if response.Body.Len() != 0 {
			t.Fatalf("%s %s: body = %q, want empty", test.method, test.path, response.Body.String())
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

func TestCreateDraftRejectsMissingVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jsonRepositoryStub{}
	handler := NewMigrationMethodHandler(service.NewMigrationMethodService(repo, service.NewMediaService(objectStoreStub{})))
	router := gin.New()
	router.POST("/api/v1/methods", handler.CreateDraftMethod)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Услуга"); err != nil {
		t.Fatal(err)
	}
	image, err := writer.CreateFormFile("image", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := image.Write([]byte("\x89PNG\r\n\x1a\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/methods", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || repo.created != nil {
		t.Fatalf("status = %d, created = %+v, body = %q", response.Code, repo.created, response.Body.String())
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
		if response.Code != http.StatusBadRequest || response.Body.Len() != 0 {
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
	router.GET("/api/methods", handler.GetGrid)
	router.GET("/api/methods/feed", handler.GetFeedItem)
	router.GET("/api/methods/feed/:id", handler.GetFeedItem)

	tests := []struct {
		path   string
		field  string
		wantID float64
	}{
		{"/api/methods", "methods", 3},
		{"/api/methods/feed", "method", 3},
		{"/api/methods/feed/7", "method", 7},
		{"/api/methods/feed/3?next=true", "method", 7},
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
		for _, field := range []string{"status", "creator_id", "created_at", "published_at"} {
			if _, exists := method[field]; exists {
				t.Fatalf("GET %s: %s was exposed in JSON", test.path, field)
			}
		}
	}
	if repo.nextID != 3 {
		t.Fatalf("next lookup ID = %d, want 3", repo.nextID)
	}
}

func TestDraftAndPublishedMethodsHaveTheSameJSONFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	description := "Готовое описание"
	timeInGb := 0.25
	reliability := 0.9
	repo := &jsonRepositoryStub{
		draft: ds.MigrationMethod{ID: 1, Title: "Черновик", Status: ds.StatusDraft},
		published: []ds.MigrationMethod{{
			ID: 2, Title: "Услуга", Status: ds.StatusPublished,
			Description: &description, TimeInGb: &timeInGb, Reliability: &reliability,
		}},
	}
	handler := NewMigrationMethodHandler(service.NewMigrationMethodService(repo))
	router := gin.New()
	router.GET("/api/methods/draft", handler.ShowAddMethodPage)
	router.GET("/api/methods/feed", handler.GetFeedItem)

	fields := []string{
		"id", "title", "description", "image_url", "video_url",
		"created_by_current_user", "likes_count", "time_in_gb", "reliability",
	}
	for _, test := range []struct {
		path            string
		key             string
		wantDescription any
		wantTime        any
		wantReliability any
	}{
		{"/api/methods/draft", "draft", nil, nil, nil},
		{"/api/methods/feed", "method", description, timeInGb, reliability},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s: status = %d", test.path, response.Code)
		}
		var body map[string]map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("GET %s: invalid JSON: %v", test.path, err)
		}
		method := body[test.key]
		if len(method) != len(fields) {
			t.Fatalf("GET %s: fields = %v", test.path, method)
		}
		for _, field := range fields {
			if _, exists := method[field]; !exists {
				t.Fatalf("GET %s: missing %s", test.path, field)
			}
		}
		if method["description"] != test.wantDescription || method["time_in_gb"] != test.wantTime || method["reliability"] != test.wantReliability {
			t.Fatalf("GET %s: optional fields = %v", test.path, method)
		}
	}
}

func TestSetLikeUsesCurrentUserAndAcceptsOnlyZeroOrOne(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jsonRepositoryStub{published: []ds.MigrationMethod{{ID: 7, Status: ds.StatusPublished}}, likesCount: 2}
	router := gin.New()
	router.POST("/api/v1/methods/:id/like", NewMigrationMethodHandler(service.NewMigrationMethodService(repo)).SetLike)

	for _, test := range []struct {
		path   string
		body   string
		status int
		value  int
		liked  bool
		count  float64
	}{
		{"/api/v1/methods/7/like", `{"like":1}`, http.StatusOK, 1, true, 3},
		{"/api/v1/methods/7/like", `{"like":1}`, http.StatusOK, 1, true, 3},
		{"/api/v1/methods/7/like", `{"like":0}`, http.StatusOK, 0, false, 2},
		{"/api/v1/methods/7/like", `{"like":2}`, http.StatusBadRequest, 0, false, 0},
		{"/api/v1/methods/7/like", `{"like":1,"status":"published"}`, http.StatusBadRequest, 0, false, 0},
		{"/api/v1/methods/7/like", `{}`, http.StatusBadRequest, 0, false, 0},
		{"/api/v1/methods/bad/like", `{"like":1}`, http.StatusBadRequest, 0, false, 0},
		{"/api/v1/methods/8/like", `{"like":1}`, http.StatusNotFound, 0, false, 0},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		router.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("POST %s %s: status = %d, body = %q", test.path, test.body, response.Code, response.Body.String())
		}
		if test.status == http.StatusOK {
			var body map[string]map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("POST %s %s: invalid JSON: %q", test.path, test.body, response.Body.String())
			}
			method := body["method"]
			if len(body) != 1 || len(method) != 3 || method["id"] != float64(7) ||
				method["is_liked"] != test.liked || method["likes_count"] != test.count {
				t.Fatalf("POST %s %s: method = %v", test.path, test.body, method)
			}
			if repo.likeID != 7 || repo.likeUser != currentCreatorID() || repo.likeValue != test.value {
				t.Fatalf("like delegation = method %d, user %d, value %d", repo.likeID, repo.likeUser, repo.likeValue)
			}
		} else if response.Body.Len() != 0 {
			t.Fatalf("POST %s %s: error body = %q", test.path, test.body, response.Body.String())
		}
	}
	if repo.likeCalls != 3 {
		t.Fatalf("SetLike repository calls = %d, want 3", repo.likeCalls)
	}
}
