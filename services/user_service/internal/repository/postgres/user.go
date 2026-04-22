package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (ur *userRepository) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
		SELECT user_id, nickname, balance, rating
		FROM users
		WHERE user_id = $1
	`

	var user domain.User
	err := ur.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.NickName,
		&user.Balance,
		&user.Rating,
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
		INSERT INTO users (user_id, nickname, balance, rating)
		VALUES ($1, $2, 0, 0)
		RETURNING user_id, nickname, balance, rating
	`

	var user domain.User
	err := ur.db.QueryRow(ctx, query, id, fmt.Sprintf("user_%d", id)).Scan(
		&user.ID,
		&user.NickName,
		&user.Balance,
		&user.Rating,
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
		RETURNING user_id, nickname, balance, rating
	`

	var updated domain.User
	err := ur.db.QueryRow(ctx, query, id, user.NickName, user.Balance).Scan(
		&updated.ID,
		&updated.NickName,
		&updated.Balance,
		&updated.Rating,
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

func (ur *userRepository) GetUserRatingHistory(ctx context.Context, userID int, params domain.UserRatingHistoryParams) (domain.UserRatingHistory, error) {
	result := domain.UserRatingHistory{
		Items: make([]domain.UserRatingHistoryPoint, 0),
	}

	err := ur.db.QueryRow(ctx, `
		SELECT rating
		FROM users
		WHERE user_id = $1
	`, userID).Scan(&result.CurrentRating)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return result, repository.ErrorUserNotFound
		}
		return result, fmt.Errorf("get current user rating: %w", err)
	}
	result.CurrentRank = domain.RankFromRating(result.CurrentRating)

	whereSQL := "user_id = $1"
	args := []any{userID}
	if params.DateFrom != nil {
		args = append(args, params.DateFrom.UTC())
		whereSQL += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}
	if params.DateTo != nil {
		args = append(args, params.DateTo.UTC())
		whereSQL += fmt.Sprintf(" AND created_at <= $%d", len(args))
	}

	rows, err := ur.db.Query(ctx, `
		SELECT
			urh.user_rating_history_id,
			urh.rounds_id,
			urh.room_id,
			urh.game_id,
			g.name_game,
			urh.source,
			urh.delta,
			urh.rating_after,
			urh.created_at
		FROM user_rating_history urh
		LEFT JOIN games g ON g.game_id = urh.game_id
		WHERE `+whereSQL+`
		ORDER BY urh.created_at ASC, urh.user_rating_history_id ASC
	`, args...)
	if err != nil {
		return result, fmt.Errorf("list user rating history: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.UserRatingHistoryPoint
		var roundID sql.NullInt64
		var roomID sql.NullInt64
		var gameID sql.NullInt64
		var gameName sql.NullString

		err := rows.Scan(
			&item.HistoryID,
			&roundID,
			&roomID,
			&gameID,
			&gameName,
			&item.Source,
			&item.Delta,
			&item.RatingAfter,
			&item.CreatedAt,
		)
		if err != nil {
			return result, fmt.Errorf("scan user rating history: %w", err)
		}

		if roundID.Valid {
			value := roundID.Int64
			item.RoundID = &value
		}
		if roomID.Valid {
			value := roomID.Int64
			item.RoomID = &value
		}
		if gameID.Valid {
			value := gameID.Int64
			item.GameID = &value
		}
		if gameName.Valid && gameName.String != "" {
			value := gameName.String
			item.GameName = &value
		}
		item.CreatedAt = item.CreatedAt.UTC()
		item.Rank = domain.RankFromRating(item.RatingAfter)
		result.PeriodChange += item.Delta
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate user rating history: %w", err)
	}

	return result, nil
}
