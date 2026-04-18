package postgres

import (
	"context"
	"errors"
	"fmt"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (ur *userRepository) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
		SELECT user_id, nickname, balance
		FROM users
		WHERE user_id = $1
	`

	var user domain.User
	err := ur.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.NickName,
		&user.Balance,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

// CreateUserByID implements [repository.UserRepository].
func (ur *userRepository) CreateUserByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
		INSERT INTO users (user_id, nickname, balance)
		VALUES ($1, $2, 0)
		RETURNING user_id, nickname, balance
	`

	var user domain.User
	err := ur.db.QueryRow(ctx, query, id, fmt.Sprintf("user_%d", id)).Scan(
		&user.ID,
		&user.NickName,
		&user.Balance,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, repository.ErrorUserAlreadyExist
		}
		return nil, fmt.Errorf("create user by id: %w", err)
	}

	return &user, nil
}

// DeleteUserByID implements [repository.UserRepository].
func (ur *userRepository) DeleteUserByID(ctx context.Context, id int) error {
	query := `
		DELETE FROM users
		WHERE user_id = $1
	`

	result, err := ur.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user by id: %w", err)
	}
	if result.RowsAffected() == 0 {
		return repository.ErrorUserNotFound
	}

	return nil
}

// UpdateUserByID implements [repository.UserRepository].
func (ur *userRepository) UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error) {
	query := `
		UPDATE users
		SET nickname = $2,
			balance = $3
		WHERE user_id = $1
		RETURNING user_id, nickname, balance
	`

	var updated domain.User
	err := ur.db.QueryRow(ctx, query, id, user.NickName, user.Balance).Scan(
		&updated.ID,
		&updated.NickName,
		&updated.Balance,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorUserNotFound
		}
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return nil, repository.ErrorNegativeBalance
		}
		return nil, fmt.Errorf("update user by id: %w", err)
	}

	return &updated, nil
}
