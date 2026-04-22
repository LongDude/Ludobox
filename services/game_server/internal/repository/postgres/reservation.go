package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"game_server/internal/domain"
	"game_server/internal/repository"

	"github.com/jackc/pgx/v5"
)

func (s *txScope) GetBalanceLocked(ctx context.Context, userID int64) (int64, error) {
	var bal int64
	err := s.tx.QueryRow(ctx, `SELECT balance FROM users WHERE user_id = $1 FOR UPDATE`, userID).Scan(&bal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("user not found: %w", err)
		}
		return 0, fmt.Errorf("lock balance: %w", err)
	}
	return bal, nil
}

func (s *txScope) UpdateBalance(ctx context.Context, userID int64, delta int64) error {
	_, err := s.tx.Exec(ctx, `UPDATE users SET balance = balance + $1 WHERE user_id = $2`, delta, userID)
	return err
}

func (s *txScope) ApplyUserRatingReward(ctx context.Context, reward domain.UserRatingReward) error {
	var ratingAfter int64
	err := s.tx.QueryRow(ctx, `
		UPDATE users
		SET rating = rating + $1
		WHERE user_id = $2
		RETURNING rating
	`, reward.Delta, reward.UserID).Scan(&ratingAfter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("update user rating: user %d not found", reward.UserID)
		}
		return fmt.Errorf("update user rating: %w", err)
	}

	_, err = s.tx.Exec(ctx, `
		INSERT INTO user_rating_history (
			user_id,
			round_participants_id,
			rounds_id,
			room_id,
			game_id,
			source,
			delta,
			rating_after
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		reward.UserID,
		reward.ParticipantID,
		reward.RoundID,
		reward.RoomID,
		reward.GameID,
		reward.Source,
		reward.Delta,
		ratingAfter,
	)
	if err != nil {
		return fmt.Errorf("insert user rating history: %w", err)
	}

	return nil
}

func (s *txScope) ReserveEntry(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	if amount <= 0 {
		return 0, repository.ErrInvalidAmount
	}
	var id int64
	err := s.tx.QueryRow(ctx, `
		INSERT INTO user_balance_reservations (round_participants_id, reservation_type, amount, status, expires_at)
		VALUES ($1, 'entry_fee', $2, 'active', $3)
		RETURNING reservation_id`,
		participantID, amount, expiresAt,
	).Scan(&id)
	return id, err
}

func (s *txScope) ReserveBoost(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	if amount <= 0 {
		return 0, repository.ErrInvalidAmount
	}
	var id int64
	err := s.tx.QueryRow(ctx, `
		INSERT INTO user_balance_reservations (round_participants_id, reservation_type, amount, status, expires_at)
		VALUES ($1, 'boost', $2, 'active', $3)
		RETURNING reservation_id`,
		participantID, amount, expiresAt,
	).Scan(&id)
	return id, err
}

func (s *txScope) ReleaseAllReservations(ctx context.Context, participantID int64) (int64, error) {
	var sum int64
	err := s.tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE user_balance_reservations
			SET status = 'released', archived_at = NOW()
			WHERE round_participants_id = $1 AND status = 'active'
			RETURNING amount
		)
		SELECT COALESCE(SUM(amount), 0) FROM updated`, participantID,
	).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("release all reservations: %w", err)
	}
	return sum, nil
}

func (s *txScope) ReleaseBoostReservations(ctx context.Context, participantID int64) (int64, error) {
	var sum int64
	err := s.tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE user_balance_reservations 
			SET status = 'released', archived_at = NOW()
			WHERE round_participants_id = $1 AND reservation_type = 'boost' AND status = 'active'
			RETURNING amount
		)
		SELECT COALESCE(SUM(amount), 0) FROM updated`, participantID,
	).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("release boost: %w", err)
	}
	if sum == 0 {
		return 0, repository.ErrActiveReservationNotFound
	}
	return sum, nil
}

func (s *txScope) CommitReservations(ctx context.Context, participantID int64) (int64, error) {
	var sum int64
	err := s.tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE user_balance_reservations 
			SET status = 'committed', archived_at = NOW()
			WHERE round_participants_id = $1 AND status = 'active'
			RETURNING amount
		)
		SELECT COALESCE(SUM(amount), 0) FROM updated`, participantID,
	).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	if sum == 0 {
		return 0, repository.ErrActiveReservationNotFound
	}
	return sum, nil
}
