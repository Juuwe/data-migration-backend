package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

type userRepositoryStub struct {
	created *ds.User
	err     error
	refresh map[string]time.Time
}

func (r *userRepositoryStub) SaveRefreshToken(_ context.Context, _ int64, hash string, ttl time.Duration) error {
	if r.refresh == nil {
		r.refresh = make(map[string]time.Time)
	}
	r.refresh[hash] = time.Now().Add(ttl)
	return nil
}

func (r *userRepositoryStub) RotateRefreshToken(_ context.Context, oldHash, newHash string, ttl time.Duration) (int64, error) {
	if _, ok := r.refresh[oldHash]; !ok {
		return 0, auth.ErrRefreshTokenNotFound
	}
	delete(r.refresh, oldHash)
	r.refresh[newHash] = time.Now().Add(ttl)
	return r.created.ID, nil
}

func (r *userRepositoryStub) IsAccessTokenBlacklisted(context.Context, string) (bool, error) {
	return false, nil
}

func (r *userRepositoryStub) RevokeTokens(context.Context, int64, string, string, time.Duration) error {
	return nil
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

func (r *userRepositoryStub) FindUserByEmail(_ context.Context, email string) (ds.User, error) {
	if r.created != nil && r.created.Email == email {
		return *r.created, nil
	}
	return ds.User{}, ds.ErrUserNotFound
}

func (r *userRepositoryStub) FindUserByID(_ context.Context, id int64) (ds.User, error) {
	if r.created != nil && r.created.ID == id {
		return *r.created, nil
	}
	return ds.User{}, ds.ErrUserNotFound
}

func userHandlerForTest(t *testing.T, repo *userRepositoryStub) *UserHandler {
	t.Helper()
	tokens, err := auth.NewManager(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	return NewUserHandler(service.NewUserService(repo, tokens, repo))
}

func TestRegisterAndLoginRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &userRepositoryStub{}
	h := userHandlerForTest(t, repo)
	router := gin.New()
	router.POST("/api/users", h.Register)
	router.POST("/api/users/login", h.Authenticate)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %q", response.Code, response.Body.String())
	}
	var registered map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &registered); err != nil || registered["id"] != float64(12) || registered["email"] != "user@example.com" {
		t.Fatalf("registration response = %q, error = %v", response.Body.String(), err)
	}
	if _, ok := registered["password"]; ok {
		t.Fatal("password exposed in registration response")
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/users/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %q", response.Code, response.Body.String())
	}
	var login struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &login); err != nil || login.AccessToken == "" || login.TokenType != "Bearer" || login.ExpiresIn != 3600 || login.RefreshToken == "" {
		t.Fatalf("login response = %q, error = %v", response.Body.String(), err)
	}
}

func TestUserRoutesRejectInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &userRepositoryStub{}
	h := userHandlerForTest(t, repo)
	router := gin.New()
	router.POST("/api/users", h.Register)
	router.POST("/api/users/login", h.Authenticate)
	for _, test := range []struct {
		path   string
		body   string
		status int
	}{
		{"/api/users", `{"id":77,"email":"user@example.com","password":"password123"}`, 400},
		{"/api/users", `{"email":"user@example.com","password":"short"}`, 400},
		{"/api/users/login", `{}`, 400},
		{"/api/users/login", `{"email":"missing@example.com","password":"password123"}`, 401},
		{"/api/users/login", `{"email":"user@example.com","password":"wrongpass"}`, 401},
		{"/api/users/login", `{"email":"user@example.com","password":"password123","admin":true}`, 400},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		router.ServeHTTP(response, request)
		if response.Code != test.status || response.Body.Len() != 0 {
			t.Fatalf("POST %s: status = %d, body = %q", test.path, response.Code, response.Body.String())
		}
	}
	if repo.created != nil {
		t.Fatal("invalid requests created a user")
	}
}

func TestRegisterDuplicateEmailReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &userRepositoryStub{err: ds.ErrUserAlreadyExists}
	router := gin.New()
	router.POST("/api/users", userHandlerForTest(t, repo).Register)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || response.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
}
