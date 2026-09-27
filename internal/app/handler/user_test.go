package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

type userRepositoryStub struct {
	created *ds.User
	err     error
}

func (r *userRepositoryStub) CreateUser(_ context.Context, user *ds.User) error {
	if r.err != nil {
		return r.err
	}
	user.ID = 12
	copy := *user
	r.created = &copy
	return nil
}

func TestUserRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &userRepositoryStub{}
	handler := NewUserHandler(service.NewUserService(repo))
	router := gin.New()
	router.POST("/api/v1/users", handler.Register)
	router.POST("/api/v1/users/login", handler.Authenticate)
	router.POST("/api/v1/users/logout", handler.Logout)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"email":"user@example.com","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %q", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["id"] != float64(12) || body["email"] != "user@example.com" {
		t.Fatalf("register response = %q, error = %v", response.Body.String(), err)
	}
	if _, ok := body["password"]; ok {
		t.Fatal("password was exposed in registration response")
	}

	for _, path := range []string{"/api/v1/users/login", "/api/v1/users/logout"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusNotImplemented || !json.Valid(response.Body.Bytes()) {
			t.Fatalf("POST %s: status = %d, body = %q", path, response.Code, response.Body.String())
		}
	}
}

func TestRegisterRejectsSystemFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &userRepositoryStub{}
	router := gin.New()
	router.POST("/api/v1/users", NewUserHandler(service.NewUserService(repo)).Register)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"id":77,"email":"user@example.com","password":"secret"}`))
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || repo.created != nil {
		t.Fatalf("status = %d, created = %+v", response.Code, repo.created)
	}
}

func TestRegisterDuplicateEmailReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &userRepositoryStub{err: ds.ErrUserAlreadyExists}
	router := gin.New()
	router.POST("/api/v1/users", NewUserHandler(service.NewUserService(repo)).Register)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"email":"user@example.com","password":"secret"}`))
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !json.Valid(response.Body.Bytes()) {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
}
