package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Juuwe/data-migration-backend/internal/ds"
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

func TestRegisterStoresPasswordAsProvided(t *testing.T) {
	repo := &userRepositoryStub{}
	svc := NewUserService(repo)

	user, err := svc.Register(context.Background(), "  student@example.com  ", "  secret  ")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.ID != 12 || user.Email != "student@example.com" || repo.created == nil {
		t.Fatalf("registered user = %+v, repository user = %+v", user, repo.created)
	}
	if user.Password != "  secret  " || repo.created.Password != "  secret  " {
		t.Fatalf("password changed: registered user = %q, repository user = %q", user.Password, repo.created.Password)
	}
}

func TestRegisterRejectsInvalidInputAndDuplicateEmail(t *testing.T) {
	repo := &userRepositoryStub{}
	svc := NewUserService(repo)
	if _, err := svc.Register(context.Background(), "bad-email", "secret"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("invalid email error = %v", err)
	}
	if _, err := svc.Register(context.Background(), "student@example.com", ""); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("empty password error = %v", err)
	}
	if repo.created != nil {
		t.Fatal("repository was called for invalid input")
	}

	repo.err = ds.ErrUserAlreadyExists
	if _, err := svc.Register(context.Background(), "student@example.com", "secret"); !errors.Is(err, ds.ErrUserAlreadyExists) {
		t.Fatalf("duplicate email error = %v", err)
	}
}
