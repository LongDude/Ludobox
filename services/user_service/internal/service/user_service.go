package service

import (
	"context"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type UserService interface {
	GetUserByID(ctx context.Context, id int) (*domain.User, error)
	CreateUserByID(ctx context.Context, id int) (*domain.User, error)
	UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error)
	UpdateUserBalance(ctx context.Context, balance_sum, id int) (*domain.User, error)
	DeleteUserByID(ctx context.Context, id int) error
}

type userService struct {
	userRepository repository.UserRepository
	logger         *logrus.Logger
}

func NewUserService(userRepository repository.UserRepository, logger *logrus.Logger) UserService {
	return &userService{
		userRepository: userRepository,
		logger:         logger,
	}
}

// CreateUserByID implements [UserService].
func (u *userService) CreateUserByID(ctx context.Context, id int) (*domain.User, error) {
	panic("unimplemented")
}

// DeleteUserByID implements [UserService].
func (u *userService) DeleteUserByID(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetUserByID implements [UserService].
func (u *userService) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	panic("unimplemented")
}

// UpdateUserBalance implements [UserService].
func (u *userService) UpdateUserBalance(ctx context.Context, balance_sum int, id int) (*domain.User, error) {
	panic("unimplemented")
}

// UpdateUserByID implements [UserService].
func (u *userService) UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error) {
	panic("unimplemented")
}
