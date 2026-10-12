package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/Juuwe/data-migration-backend/pkg/middleware"
	"github.com/gin-gonic/gin"
)

type authUserRepo struct {
	id  int64
	err error
}

type authFlowRepo struct {
	service.MigrationMethodRepository
	user           ds.User
	draftCreatorID int64
	refresh        map[string]time.Time
	blacklist      map[string]bool
}

func (r *authFlowRepo) SaveRefreshToken(_ context.Context, _ int64, hash string, ttl time.Duration) error {
	if r.refresh == nil {
		r.refresh = make(map[string]time.Time)
	}
	r.refresh[hash] = time.Now().Add(ttl)
	return nil
}
func (r *authFlowRepo) RotateRefreshToken(_ context.Context, oldHash, newHash string, ttl time.Duration) (int64, error) {
	if _, ok := r.refresh[oldHash]; !ok {
		return 0, auth.ErrRefreshTokenNotFound
	}
	delete(r.refresh, oldHash)
	r.refresh[newHash] = time.Now().Add(ttl)
	return r.user.ID, nil
}
func (r *authFlowRepo) IsAccessTokenBlacklisted(_ context.Context, hash string) (bool, error) {
	return r.blacklist[hash], nil
}
func (r *authFlowRepo) RevokeTokens(_ context.Context, userID int64, accessHash, refreshHash string, _ time.Duration) error {
	if userID != r.user.ID || !r.refresh[refreshHash].After(time.Now()) {
		return auth.ErrRefreshTokenNotFound
	}
	delete(r.refresh, refreshHash)
	if r.blacklist == nil {
		r.blacklist = make(map[string]bool)
	}
	r.blacklist[accessHash] = true
	return nil
}

func (r *authFlowRepo) CreateUser(_ context.Context, user *ds.User) error {
	user.ID = 9
	r.user = *user
	return nil
}
func (r *authFlowRepo) FindUserByEmail(_ context.Context, email string) (ds.User, error) {
	if r.user.ID != 0 && r.user.Email == email {
		return r.user, nil
	}
	return ds.User{}, ds.ErrUserNotFound
}
func (r *authFlowRepo) FindUserByID(_ context.Context, id int64) (ds.User, error) {
	if r.user.ID == id {
		return r.user, nil
	}
	return ds.User{}, ds.ErrUserNotFound
}
func (r *authFlowRepo) FindDraft(_ context.Context, creatorID int64) (ds.MigrationMethod, error) {
	r.draftCreatorID = creatorID
	return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
}

func (*authUserRepo) CreateUser(context.Context, *ds.User) error { return nil }
func (*authUserRepo) SaveRefreshToken(context.Context, int64, string, time.Duration) error {
	return nil
}
func (*authUserRepo) RotateRefreshToken(context.Context, string, string, time.Duration) (int64, error) {
	return 0, auth.ErrRefreshTokenNotFound
}
func (*authUserRepo) IsAccessTokenBlacklisted(context.Context, string) (bool, error) {
	return false, nil
}
func (*authUserRepo) RevokeTokens(context.Context, int64, string, string, time.Duration) error {
	return auth.ErrRefreshTokenNotFound
}
func (*authUserRepo) FindUserByEmail(context.Context, string) (ds.User, error) {
	return ds.User{}, ds.ErrUserNotFound
}
func (r *authUserRepo) FindUserByID(_ context.Context, id int64) (ds.User, error) {
	if r.err != nil {
		return ds.User{}, r.err
	}
	if id != r.id {
		return ds.User{}, ds.ErrUserNotFound
	}
	return ds.User{ID: id}, nil
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens, err := auth.NewManager(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := &authUserRepo{id: 7}
	users := service.NewUserService(repo, tokens, repo)
	router := gin.New()
	respond := func(c *gin.Context) { c.String(http.StatusOK, fmt.Sprint(auth.UserID(c.Request.Context()))) }
	router.GET("/public", middleware.Auth(userAuthenticator(users), false), respond)
	router.GET("/private", middleware.Auth(userAuthenticator(users), true), respond)
	token, _, err := tokens.Issue(7)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path   string
		header string
		status int
		body   string
	}{
		{"/public", "", 200, "0"},
		{"/private", "", 401, ""},
		{"/private", "Basic abc", 401, ""},
		{"/private", "Bearer", 401, ""},
		{"/private", "Bearer " + token, 200, "7"},
		{"/public", "Bearer " + token, 200, "7"},
		{"/public", "Bearer bad", 401, ""},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		if test.header != "" {
			request.Header.Set("Authorization", test.header)
		}
		router.ServeHTTP(response, request)
		if response.Code != test.status || response.Body.String() != test.body {
			t.Fatalf("GET %s with %q: status = %d, body = %q", test.path, test.header, response.Code, response.Body.String())
		}
	}
	repo.id = 8
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatalf("deleted user status = %d", response.Code)
	}
	repo.err = errors.New("database unavailable")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 500 {
		t.Fatalf("database error status = %d", response.Code)
	}
}

func TestRegisterLoginAndProtectedRoute(t *testing.T) {
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
	tokens, err := auth.NewManager(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := &authFlowRepo{}
	router := NewRouter(service.NewMigrationMethodService(repo), service.NewUserService(repo, tokens, repo))
	request := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"email":"new@example.com","password":"password123"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d", response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/methods/draft", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated draft status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/users/login", strings.NewReader(`{"email":"new@example.com","password":"password123"}`))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %q", response.Code, response.Body.String())
	}
	var login struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &login); err != nil || login.AccessToken == "" || login.RefreshToken == "" || login.ExpiresIn != 3600 {
		t.Fatalf("login response = %q, error = %v", response.Body.String(), err)
	}
	request = httptest.NewRequest(http.MethodGet, "/api/methods/draft", nil)
	request.Header.Set("Authorization", "Bearer "+login.AccessToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || repo.draftCreatorID != 9 {
		t.Fatalf("authenticated draft status = %d, creator ID = %d", response.Code, repo.draftCreatorID)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/users/refresh", strings.NewReader(`{}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing refresh token status = %d", response.Code)
	}
	refreshBody := `{"refresh_token":"` + login.RefreshToken + `"}`
	request = httptest.NewRequest(http.MethodPost, "/api/users/refresh", strings.NewReader(refreshBody))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %q", response.Code, response.Body.String())
	}
	var refreshed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &refreshed); err != nil || refreshed.AccessToken == login.AccessToken || refreshed.RefreshToken == login.RefreshToken || refreshed.ExpiresIn != 3600 {
		t.Fatalf("refresh response = %q, error = %v", response.Body.String(), err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/users/refresh", strings.NewReader(refreshBody)))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh token status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/api/methods/draft", nil)
	request.Header.Set("Authorization", "Bearer "+refreshed.AccessToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("refreshed access token status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/users/logout", strings.NewReader(`{"refresh_token":"`+refreshed.RefreshToken+`"}`))
	request.Header.Set("Authorization", "Bearer "+refreshed.AccessToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/api/methods/draft", nil)
	request.Header.Set("Authorization", "Bearer "+refreshed.AccessToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked access token status = %d", response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/users/refresh", strings.NewReader(`{"refresh_token":"`+refreshed.RefreshToken+`"}`)))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked refresh token status = %d", response.Code)
	}
}
