package postgres

import (
	"context"
	"fmt"
	"user_service/internal/domain"
)

func (ur *userRepository) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	return nil, fmt.Errorf("not impl")
}

// CreateUserByID implements [repository.UserRepository].
func (ur *userRepository) CreateUserByID(ctx context.Context, id int) (*domain.User, error) {
	return nil, fmt.Errorf("not impl")
}

// DeleteUserByID implements [repository.UserRepository].
func (ur *userRepository) DeleteUserByID(ctx context.Context, id int) error {
	return fmt.Errorf("not impl")
}

// UpdateUserByID implements [repository.UserRepository].
func (ur *userRepository) UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error) {
	return nil, fmt.Errorf("not impl")
}
