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
	GetUserRatingHistory(ctx context.Context, userID int, params domain.UserRatingHistoryParams) (domain.UserRatingHistory, error)
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
	user, err := u.userRepository.CreateUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return enrichUserRank(user), nil
}

// DeleteUserByID implements [UserService].
func (u *userService) DeleteUserByID(ctx context.Context, id int) error {
	return u.userRepository.DeleteUserByID(ctx, id)
}

// GetUserByID implements [UserService].
func (u *userService) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	user, err := u.userRepository.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return enrichUserRank(user), nil
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

	updated, err := u.userRepository.UpdateUserByID(ctx, id, user)
	if err != nil {
		return nil, err
	}
	return enrichUserRank(updated), nil
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

	updated, err := u.userRepository.UpdateUserByID(ctx, id, current)
	if err != nil {
		return nil, err
	}
	return enrichUserRank(updated), nil
}

func (u *userService) GetUserRatingHistory(ctx context.Context, userID int, params domain.UserRatingHistoryParams) (domain.UserRatingHistory, error) {
	return u.userRepository.GetUserRatingHistory(ctx, userID, params)
}

func enrichUserRank(user *domain.User) *domain.User {
	if user == nil {
		return nil
	}
	user.Rank = domain.RankFromRating(user.Rating)
	return user
}
