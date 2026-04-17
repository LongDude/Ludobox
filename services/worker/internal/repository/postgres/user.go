package postgres

import (
	"context"
	"errors"
	"fmt"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5"
)

func (ur *adminRepository) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
        SELECT id, first_name, last_name, email, email_confirmed, pass_hash,
               google_id, yandex_id, vk_id, photo, roles, locale
        FROM users
        WHERE id = $1 AND is_active = true
    `

	var user domain.User
	var passHash []byte
	err := ur.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.EmailConfirmed,
		&passHash,
		&user.GoogleID,
		&user.YandexID,
		&user.VkID,
		&user.Photo,
		&user.Roles,
		&user.LocaleType,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	if len(passHash) > 0 {
		s := string(passHash)
		user.Password = &s
	}

	return &user, nil
}
