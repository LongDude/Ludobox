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
	return u.userRepository.CreateUserByID(ctx, id)
}

// DeleteUserByID implements [UserService].
func (u *userService) DeleteUserByID(ctx context.Context, id int) error {
	return u.userRepository.DeleteUserByID(ctx, id)
}

// GetUserByID implements [UserService].
func (u *userService) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	return u.userRepository.GetUserByID(ctx, id)
}

// UpdateUserBalance implements [UserService].
func (u *userService) UpdateUserBalance(ctx context.Context, balance_sum int, id int) (*domain.User, error) {
	user, err := u.userRepository.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Balance += balance_sum
	if user.Balance < 0 {
		return nil, repository.ErrorNegativeBalance
	}

	return u.userRepository.UpdateUserByID(ctx, id, user)
}

// UpdateUserByID implements [UserService].
func (u *userService) UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error) {
	current, err := u.userRepository.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if user.NickName != "" {
		current.NickName = user.NickName
	}
	if user.Balance != 0 || current.Balance == 0 {
		current.Balance = user.Balance
	}

	return u.userRepository.UpdateUserByID(ctx, id, current)
}
