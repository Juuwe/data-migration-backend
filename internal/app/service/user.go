package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/Juuwe/data-migration-backend/internal/ds"
)

var (
	ErrInvalidEmail    = errors.New("некорректный email")
	ErrInvalidPassword = errors.New("некорректный пароль")
	ErrNotImplemented  = errors.New("будет реализовано в четвертой лабораторной")
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *ds.User) error
}

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, email, password string) (ds.User, error) {
	email = strings.TrimSpace(email)
	address, err := mail.ParseAddress(email)
	if email == "" || len(email) > 255 || err != nil || address.Address != email {
		return ds.User{}, ErrInvalidEmail
	}
	if password == "" {
		return ds.User{}, ErrInvalidPassword
	}

	user := ds.User{Email: email, Password: password}
	if err := s.repo.CreateUser(ctx, &user); err != nil {
		return ds.User{}, err
	}
	return user, nil
}

func (s *UserService) Authenticate(context.Context) error {
	return ErrNotImplemented
}

func (s *UserService) Logout(context.Context) error {
	return ErrNotImplemented
}
