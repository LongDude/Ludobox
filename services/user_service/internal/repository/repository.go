package repository

import (
	"context"
	"errors"
	"user_service/internal/domain"
)

var (
	ErrorUserNotFound     = errors.New("user not found")
	ErrorUserAlreadyExist = errors.New("user already exist")
)

type UserRepository interface {
	GetUserByID(ctx context.Context, id int) (*domain.User, error)
	CreateUserByID(ctx context.Context, id int) (*domain.User, error)
	UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error)
	DeleteUserByID(ctx context.Context, id int) error
}
type SessionRepository interface {
}
