package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Juuwe/data-migration-backend/internal/ds"
)

type repositoryStub struct {
	draft         ds.MigrationMethod
	draftErr      error
	method        ds.MigrationMethod
	methodErr     error
	published     []ds.MigrationMethod
	publishedErr  error
	created       *ds.MigrationMethod
	updated       *ds.MigrationMethod
	deleteID      int64
	createErr     error
	updateErr     error
	softDeleteErr error
}

func (r *repositoryStub) FindByID(context.Context, int64) (ds.MigrationMethod, error) {
	return r.method, r.methodErr
}

func (r *repositoryStub) FindNextPublishedAfterID(context.Context, int64) (ds.MigrationMethod, error) {
	return r.method, r.methodErr
}

func (r *repositoryStub) FindDraft(context.Context, int64) (ds.MigrationMethod, error) {
	return r.draft, r.draftErr
}

func (r *repositoryStub) FindPublishedByTime(context.Context, float64, float64) ([]ds.MigrationMethod, error) {
	return nil, nil
}

func (r *repositoryStub) FindPublished(context.Context) ([]ds.MigrationMethod, error) {
	return r.published, r.publishedErr
}

func (r *repositoryStub) CountMethodLikesByID(context.Context, int64) (int, error) {
	return 0, nil
}

func (r *repositoryStub) CountLikesByMethodIDs(context.Context, []int64) (map[int64]int, error) {
	return map[int64]int{}, nil
}

func (r *repositoryStub) Create(_ context.Context, method *ds.MigrationMethod) error {
	copy := *method
	r.created = &copy
	return r.createErr
}

func (r *repositoryStub) Update(_ context.Context, method *ds.MigrationMethod) error {
	copy := *method
	r.updated = &copy
	return r.updateErr
}

func (r *repositoryStub) SoftDeleteSQL(_ context.Context, id int64) error {
	r.deleteID = id
	return r.softDeleteErr
}

func TestGetDraftReturnsMissingDraftWithoutError(t *testing.T) {
	repo := &repositoryStub{draftErr: ds.ErrMigrationMethodNotFound}
	svc := NewMigrationMethodService(repo)

	_, exists, err := svc.GetDraft(context.Background(), 1)

	if err != nil {
		t.Fatalf("GetDraft() error = %v", err)
	}
	if exists {
		t.Fatal("GetDraft() exists = true, want false")
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

	view, err := svc.GetByID(context.Background(), 7)

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

	_, err := svc.GetByID(context.Background(), 7)

	if err == nil {
		t.Fatal("GetByID() error = nil, want deleted method error")
	}
}

func TestGetNextPublishedAfterLastWrapsToFirst(t *testing.T) {
	repo := &repositoryStub{
		methodErr: ds.ErrMigrationMethodNotFound,
		published: []ds.MigrationMethod{{ID: 3, Status: ds.StatusPublished}},
	}
	svc := NewMigrationMethodService(repo)

	view, err := svc.GetNextPublishedAfterID(context.Background(), 5)

	if err != nil {
		t.Fatalf("GetNextPublishedAfterID() error = %v", err)
	}
	if view.ID != 3 {
		t.Fatalf("GetNextPublishedAfterID() ID = %d, want 3", view.ID)
	}
}

func TestGetPublishedUsesStoredMediaURLs(t *testing.T) {
	repo := &repositoryStub{
		published: []ds.MigrationMethod{
			{
				ID:       1,
				Status:   ds.StatusPublished,
				ImageURL: "http://localhost:9000/data-migration-service/first.jpg",
				VideoURL: "http://localhost:9000/data-migration-service/first.mp4",
			},
			{ID: 2, Status: ds.StatusPublished},
		},
	}
	svc := NewMigrationMethodService(repo)

	views, err := svc.GetPublished(context.Background())

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
	svc := NewMigrationMethodService(repo)

	err := svc.CreateDraftMethod(context.Background(), "  Быстрая миграция  ", 3)

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
}

func TestCreateDraftMethodRejectsSecondDraft(t *testing.T) {
	repo := &repositoryStub{draft: ds.MigrationMethod{ID: 1, Status: ds.StatusDraft}}
	svc := NewMigrationMethodService(repo)

	err := svc.CreateDraftMethod(context.Background(), "Еще один", 1)

	if err == nil {
		t.Fatal("CreateDraftMethod() error = nil, want duplicate draft error")
	}
	if repo.created != nil {
		t.Fatal("repository Create() was called for a duplicate draft")
	}
}

func TestPublishDraft(t *testing.T) {
	repo := &repositoryStub{draft: ds.MigrationMethod{ID: 2, Status: ds.StatusDraft}}
	svc := NewMigrationMethodService(repo)

	err := svc.PublishDraft(context.Background(), 1, "  Проверенное описание  ", 0.25, 0.999)

	if err != nil {
		t.Fatalf("PublishDraft() error = %v", err)
	}
	if repo.updated == nil {
		t.Fatal("repository Update() was not called")
	}
	if repo.updated.Status != ds.StatusPublished {
		t.Errorf("updated status = %q", repo.updated.Status)
	}
	if repo.updated.Description != "Проверенное описание" {
		t.Errorf("updated description = %q", repo.updated.Description)
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

			err := svc.PublishDraft(context.Background(), 1, tt.description, tt.timeInGB, tt.reliability)

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

	if err := svc.DeleteMethod(context.Background(), 9); err != nil {
		t.Fatalf("DeleteMethod() error = %v", err)
	}
	if repo.deleteID != 9 {
		t.Fatalf("SoftDeleteSQL() id = %d, want 9", repo.deleteID)
	}
}
