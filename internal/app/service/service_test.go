package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Juuwe/data-migration-backend/internal/ds"
)

type objectStoreStub struct {
	keys    []string
	deleted []string
}

func (s *objectStoreStub) Put(_ context.Context, key string, body io.ReadSeeker, size int64, contentType string) error {
	if size <= 0 || contentType == "" {
		return errors.New("invalid upload")
	}
	s.keys = append(s.keys, key)
	return nil
}

func (s *objectStoreStub) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *objectStoreStub) PublicURL(key string) string {
	return "http://localhost:9000/data-migration-service/" + key
}

func mediaInputs() (MediaInput, MediaInput) {
	image := []byte("\x89PNG\r\n\x1a\n")
	video := []byte("\x1a\x45\xdf\xa3")
	return MediaInput{Reader: bytes.NewReader(image), Size: int64(len(image))},
		MediaInput{Reader: bytes.NewReader(video), Size: int64(len(video))}
}

type repositoryStub struct {
	draft              ds.MigrationMethod
	draftErr           error
	method             ds.MigrationMethod
	methodErr          error
	published          []ds.MigrationMethod
	publishedErr       error
	created            *ds.MigrationMethod
	updated            *ds.MigrationMethod
	deleteID           int64
	deleteCreatorID    int64
	likeMethodID       int64
	likeUserID         int64
	likeValue          int
	likeCalls          int
	likesCount         int
	isLiked            bool
	createErr          error
	updateErr          error
	softDeleteErr      error
	findDraftCreatorID int64
}

func (r *repositoryStub) FindByID(context.Context, int64) (ds.MigrationMethod, error) {
	return r.method, r.methodErr
}

func (r *repositoryStub) FindNextPublishedAfterID(context.Context, int64) (ds.MigrationMethod, error) {
	return r.method, r.methodErr
}

func (r *repositoryStub) FindDraft(_ context.Context, creatorID int64) (ds.MigrationMethod, error) {
	r.findDraftCreatorID = creatorID
	return r.draft, r.draftErr
}

func (r *repositoryStub) FindPublishedByTime(context.Context, float64, float64) ([]ds.MigrationMethod, error) {
	return nil, nil
}

func (r *repositoryStub) FindPublished(context.Context) ([]ds.MigrationMethod, error) {
	return r.published, r.publishedErr
}

func (r *repositoryStub) CountMethodLikesByID(context.Context, int64) (int, error) {
	return r.likesCount, nil
}

func (r *repositoryStub) CountLikesByMethodIDs(context.Context, []int64) (map[int64]int, error) {
	return map[int64]int{}, nil
}

func (r *repositoryStub) Create(_ context.Context, method *ds.MigrationMethod) error {
	if r.createErr == nil {
		method.ID = 7
	}
	copy := *method
	r.created = &copy
	return r.createErr
}

func (r *repositoryStub) Update(_ context.Context, method *ds.MigrationMethod) error {
	copy := *method
	r.updated = &copy
	return r.updateErr
}

func (r *repositoryStub) SoftDeleteSQL(_ context.Context, id, creatorID int64) error {
	r.deleteID = id
	r.deleteCreatorID = creatorID
	return r.softDeleteErr
}

func (r *repositoryStub) SetLike(_ context.Context, methodID, userID int64, like int) error {
	r.likeMethodID = methodID
	r.likeUserID = userID
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

func TestGetDraftReturnsMissingDraftWithoutError(t *testing.T) {
	repo := &repositoryStub{draftErr: ds.ErrMigrationMethodNotFound}
	svc := NewMigrationMethodService(repo)

	_, exists, err := svc.GetDraft(context.Background(), 3)

	if err != nil {
		t.Fatalf("GetDraft() error = %v", err)
	}
	if exists {
		t.Fatal("GetDraft() exists = true, want false")
	}
	if repo.findDraftCreatorID != 3 {
		t.Fatalf("FindDraft() creator ID = %d, want 3", repo.findDraftCreatorID)
	}
}

func TestGetDraftPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database is unavailable")
	repo := &repositoryStub{draftErr: wantErr}
	svc := NewMigrationMethodService(repo)

	_, _, err := svc.GetDraft(context.Background(), 1)

	if !errors.Is(err, wantErr) {
		t.Fatalf("GetDraft() error = %v, want %v", err, wantErr)
	}
}

func TestGetByIDUsesDefaultMedia(t *testing.T) {
	repo := &repositoryStub{method: ds.MigrationMethod{ID: 7, Status: ds.StatusPublished}}
	svc := NewMigrationMethodService(repo)

	view, err := svc.GetByID(context.Background(), 7, 1)

	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if view.ImageURL != DefaultImageURL {
		t.Errorf("ImageURL = %q, want %q", view.ImageURL, DefaultImageURL)
	}
	if view.VideoURL != DefaultVideoURL {
		t.Errorf("VideoURL = %q, want %q", view.VideoURL, DefaultVideoURL)
	}
}

func TestGetByIDRejectsDeletedMethod(t *testing.T) {
	repo := &repositoryStub{method: ds.MigrationMethod{ID: 7, Status: ds.StatusDeleted}}
	svc := NewMigrationMethodService(repo)

	_, err := svc.GetByID(context.Background(), 7, 1)

	if err == nil {
		t.Fatal("GetByID() error = nil, want deleted method error")
	}
}

func TestGetByIDRejectsDraftMethod(t *testing.T) {
	repo := &repositoryStub{method: ds.MigrationMethod{ID: 7, Status: ds.StatusDraft}}
	svc := NewMigrationMethodService(repo)

	_, err := svc.GetByID(context.Background(), 7, 1)

	if !errors.Is(err, ds.ErrMigrationMethodNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, ds.ErrMigrationMethodNotFound)
	}
}

func TestGetNextPublishedAfterLastWrapsToFirst(t *testing.T) {
	repo := &repositoryStub{
		methodErr: ds.ErrMigrationMethodNotFound,
		published: []ds.MigrationMethod{{ID: 3, Status: ds.StatusPublished}},
	}
	svc := NewMigrationMethodService(repo)

	view, err := svc.GetNextPublishedAfterID(context.Background(), 5, 1)

	if err != nil {
		t.Fatalf("GetNextPublishedAfterID() error = %v", err)
	}
	if view.ID != 3 {
		t.Fatalf("GetNextPublishedAfterID() ID = %d, want 3", view.ID)
	}
}

func TestGetPublishedBuildsMediaURLsFromKeys(t *testing.T) {
	repo := &repositoryStub{
		published: []ds.MigrationMethod{
			{
				ID:       1,
				Status:   ds.StatusPublished,
				ImageKey: "first.jpg",
				VideoKey: "first.mp4",
			},
			{ID: 2, Status: ds.StatusPublished},
		},
	}
	svc := NewMigrationMethodService(repo, NewMediaService(&objectStoreStub{}))

	views, err := svc.GetPublished(context.Background(), 1)

	if err != nil {
		t.Fatalf("GetPublished() error = %v", err)
	}
	if views[0].ImageURL != "http://localhost:9000/data-migration-service/first.jpg" || views[0].VideoURL != "http://localhost:9000/data-migration-service/first.mp4" {
		t.Errorf("first view URLs = %q, %q", views[0].ImageURL, views[0].VideoURL)
	}
	if views[1].ImageURL != DefaultImageURL || views[1].VideoURL != DefaultVideoURL {
		t.Errorf("missing media URLs = %q, %q; want defaults", views[1].ImageURL, views[1].VideoURL)
	}
}

func TestCreateDraftMethod(t *testing.T) {
	repo := &repositoryStub{draftErr: ds.ErrMigrationMethodNotFound}
	store := &objectStoreStub{}
	svc := NewMigrationMethodService(repo, NewMediaService(store))
	image, video := mediaInputs()

	view, err := svc.CreateDraftMethod(context.Background(), "  Быстрая миграция  ", 3, image, video)

	if err != nil {
		t.Fatalf("CreateDraftMethod() error = %v", err)
	}
	if repo.created == nil {
		t.Fatal("repository Create() was not called")
	}
	if repo.created.Title != "Быстрая миграция" {
		t.Errorf("created title = %q", repo.created.Title)
	}
	if repo.created.Status != ds.StatusDraft || repo.created.CreatorID != 3 {
		t.Errorf("created method = %+v", *repo.created)
	}
	if !strings.HasPrefix(repo.created.ImageKey, "images/") || !strings.HasSuffix(repo.created.ImageKey, ".png") ||
		!strings.HasPrefix(repo.created.VideoKey, "videos/") || !strings.HasSuffix(repo.created.VideoKey, ".webm") || len(store.keys) != 2 {
		t.Errorf("media keys = %q, %q; uploaded = %v", repo.created.ImageKey, repo.created.VideoKey, store.keys)
	}
	if repo.created.Description != nil || repo.created.TimeInGb != nil || repo.created.Reliability != nil || repo.created.PublishedAt != nil {
		t.Errorf("draft fields must be nil: %+v", *repo.created)
	}
	if view.ID != repo.created.ID || view.Title != repo.created.Title ||
		view.Description != nil || view.TimeInGb != nil || view.Reliability != nil ||
		view.ImageURL != store.PublicURL(repo.created.ImageKey) || view.VideoURL != store.PublicURL(repo.created.VideoKey) {
		t.Errorf("created view = %+v", view)
	}
}

func TestCreateDraftMethodRejectsSecondDraft(t *testing.T) {
	repo := &repositoryStub{draft: ds.MigrationMethod{ID: 1, Status: ds.StatusDraft}}
	svc := NewMigrationMethodService(repo)

	image, video := mediaInputs()
	_, err := svc.CreateDraftMethod(context.Background(), "Еще один", 1, image, video)

	if err == nil {
		t.Fatal("CreateDraftMethod() error = nil, want duplicate draft error")
	}
	if repo.created != nil {
		t.Fatal("repository Create() was called for a duplicate draft")
	}
}

func TestCreateDraftMethodRequiresMedia(t *testing.T) {
	repo := &repositoryStub{draftErr: ds.ErrMigrationMethodNotFound}
	svc := NewMigrationMethodService(repo, NewMediaService(&objectStoreStub{}))
	image, _ := mediaInputs()
	if _, err := svc.CreateDraftMethod(context.Background(), "Услуга", 1, image, MediaInput{}); !errors.Is(err, ErrInvalidMedia) {
		t.Fatalf("error = %v, want ErrInvalidMedia", err)
	}
	if repo.created != nil {
		t.Fatal("draft was saved without video")
	}
}

func TestCreateDraftMethodCleansUpAfterDatabaseError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &repositoryStub{draftErr: ds.ErrMigrationMethodNotFound, createErr: wantErr}
	store := &objectStoreStub{}
	svc := NewMigrationMethodService(repo, NewMediaService(store))
	image, video := mediaInputs()
	if _, err := svc.CreateDraftMethod(context.Background(), "Услуга", 1, image, video); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want database error", err)
	}
	if len(store.keys) != 2 || len(store.deleted) != 2 || store.deleted[0] != store.keys[0] || store.deleted[1] != store.keys[1] {
		t.Fatalf("uploaded = %v, deleted = %v", store.keys, store.deleted)
	}
}

func TestPublishDraft(t *testing.T) {
	repo := &repositoryStub{draft: ds.MigrationMethod{ID: 2, Status: ds.StatusDraft}}
	svc := NewMigrationMethodService(repo)

	view, err := svc.PublishDraft(context.Background(), 3, "  Проверенное описание  ", 0.25, 0.999)

	if err != nil {
		t.Fatalf("PublishDraft() error = %v", err)
	}
	if repo.updated == nil {
		t.Fatal("repository Update() was not called")
	}
	if repo.updated.Status != ds.StatusPublished {
		t.Errorf("updated status = %q", repo.updated.Status)
	}
	if repo.findDraftCreatorID != 3 {
		t.Errorf("FindDraft() creator ID = %d, want 3", repo.findDraftCreatorID)
	}
	if repo.updated.Description == nil {
		t.Fatal("updated description is nil")
	}
	if *repo.updated.Description != "Проверенное описание" {
		t.Errorf("updated description = %q", *repo.updated.Description)
	}
	if view.ID != repo.updated.ID || view.Title != repo.updated.Title ||
		view.Description == nil || *view.Description != "Проверенное описание" ||
		view.TimeInGb == nil || *view.TimeInGb != 0.25 ||
		view.Reliability == nil || *view.Reliability != 0.999 {
		t.Errorf("published view = %+v", view)
	}
}

func TestPublishDraftValidatesFields(t *testing.T) {
	tests := []struct {
		name        string
		description string
		timeInGB    float64
		reliability float64
	}{
		{name: "empty description", timeInGB: 0.1, reliability: 0.9},
		{name: "zero time", description: "Описание", reliability: 0.9},
		{name: "reliability above one", description: "Описание", timeInGB: 0.1, reliability: 1.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &repositoryStub{draft: ds.MigrationMethod{ID: 2, Status: ds.StatusDraft}}
			svc := NewMigrationMethodService(repo)

			_, err := svc.PublishDraft(context.Background(), 1, tt.description, tt.timeInGB, tt.reliability)

			if err == nil {
				t.Fatal("PublishDraft() error = nil, want validation error")
			}
			if repo.updated != nil {
				t.Fatal("repository Update() was called for invalid fields")
			}
		})
	}
}

func TestDeleteMethodUsesRepositorySQLMethod(t *testing.T) {
	repo := &repositoryStub{}
	svc := NewMigrationMethodService(repo)

	if err := svc.DeleteMethod(context.Background(), 9, 3); err != nil {
		t.Fatalf("DeleteMethod() error = %v", err)
	}
	if repo.deleteID != 9 {
		t.Fatalf("SoftDeleteSQL() id = %d, want 9", repo.deleteID)
	}
	if repo.deleteCreatorID != 3 {
		t.Fatalf("SoftDeleteSQL() creator ID = %d, want 3", repo.deleteCreatorID)
	}
}

func TestSetLikeAllowsOnlyPublishedMethodsAndValidValues(t *testing.T) {
	repo := &repositoryStub{method: ds.MigrationMethod{ID: 7, Status: ds.StatusPublished}, likesCount: 2}
	svc := NewMigrationMethodService(repo)

	for _, test := range []struct {
		value int
		liked bool
		count int
	}{
		{1, true, 3},
		{1, true, 3},
		{0, false, 2},
	} {
		view, err := svc.SetLike(context.Background(), 7, 3, test.value)
		if err != nil {
			t.Fatalf("SetLike(%d) error = %v", test.value, err)
		}
		if view.ID != 7 || view.IsLiked != test.liked || view.LikesCount != test.count {
			t.Fatalf("SetLike(%d) view = %+v", test.value, view)
		}
		if repo.likeMethodID != 7 || repo.likeUserID != 3 || repo.likeValue != test.value {
			t.Fatalf("SetLike(%d) delegated method=%d user=%d value=%d", test.value, repo.likeMethodID, repo.likeUserID, repo.likeValue)
		}
	}
	if repo.likeCalls != 3 {
		t.Fatalf("SetLike() calls = %d, want 3", repo.likeCalls)
	}

	if _, err := svc.SetLike(context.Background(), 7, 3, 2); !errors.Is(err, ErrInvalidLike) {
		t.Fatalf("SetLike(2) error = %v, want invalid like", err)
	}
	repo.method.Status = ds.StatusDeleted
	if _, err := svc.SetLike(context.Background(), 7, 3, 1); !errors.Is(err, ds.ErrMigrationMethodNotFound) {
		t.Fatalf("SetLike(deleted) error = %v, want not found", err)
	}
	if repo.likeCalls != 3 {
		t.Fatalf("SetLike() calls after invalid requests = %d, want 3", repo.likeCalls)
	}
}
