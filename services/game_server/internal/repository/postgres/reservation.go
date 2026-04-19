package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInsufficientBalance       = errors.New("insufficient balance") 
	ErrActiveReservationNotFound = errors.New("active reservation not found") // can be ignored
	ErrInvalidAmount             = errors.New("invalid amount")
)

// ReserveBalance performs a transaction that checks the user balance,
// decrements it if enough funds exist, and creates an active reservation.
func ReserveBalance(
	ctx context.Context,
	db *pgxpool.Pool,
	userID int64,
	roomID int64,
	reservationType string,
	amount int64,
	idempotencyKey string,
	expiresAt time.Time,
) (int64, error) {
	if amount <= 0 {
		return 0, ErrInvalidAmount
	}

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin reserve balance transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var balance int64
	err = tx.QueryRow(ctx,
		`SELECT balance FROM users WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("reserve balance: %w", err)
		}
		return 0, fmt.Errorf("reserve balance: %w", err)
	}

	if balance < amount {
		return 0, ErrInsufficientBalance
	}

	_, err = tx.Exec(ctx,
		`UPDATE users SET balance = balance - $1 WHERE user_id = $2`,
		amount,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("reserve balance: decrease user balance: %w", err)
	}

	var reservationID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO user_balance_reservations (
            user_id,
            room_id,
            reservation_type,
            amount,
            status,
            idempotency_key,
            expires_at 
        ) VALUES ($1, $2, $3, $4, 'active', $5, $6)
        RETURNING reservation_id`,
		userID,
		roomID,
		reservationType,
		amount,
		idempotencyKey,
		expiresAt,
	).Scan(&reservationID)
	if err != nil {
		return 0, fmt.Errorf("reserve balance: create reservation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("reserve balance: commit transaction: %w", err)
	}

	return reservationID, nil
}

// ReleaseReservations cancels active reservations for a user in a room and
// refunds the total amount back to the user balance.
func ReleaseReservations(
	ctx context.Context,
	db *pgxpool.Pool,
	userID int64,
	roomID int64,
) (int64, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin release reservation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var refundAmount int64
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM user_balance_reservations
         WHERE user_id = $1 AND room_id = $2 AND status = 'active'
         FOR UPDATE`,
		userID,
		roomID,
	).Scan(&refundAmount)
	if err != nil {
		return 0, fmt.Errorf("release reservation: query active reservations: %w", err)
	}

	if refundAmount == 0 {
		return 0, ErrActiveReservationNotFound
	}

	_, err = tx.Exec(ctx,
		`UPDATE user_balance_reservations
         SET status = 'released', archived_at = NOW(), released_at = NOW()
         WHERE user_id = $1 AND room_id = $2 AND status = 'active'`,
		userID,
		roomID,
	)
	if err != nil {
		return 0, fmt.Errorf("release reservation: update reservation status: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE users SET balance = balance + $1 WHERE user_id = $2`,
		refundAmount,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("release reservation: refund user balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("release reservation: commit transaction: %w", err)
	}

	return refundAmount, nil
}

// CommitReservations finalizes active reservations for a room without awarding any win amount.
func CommitReservations(
	ctx context.Context,
	db *pgxpool.Pool,
	userID int64,
	roomID int64,
) (int64, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin commit reservation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// can be deleted
	var committedAmount int64
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM user_balance_reservations
         WHERE user_id = $1 AND room_id = $2 AND status = 'active'
         FOR UPDATE`,
		userID,
		roomID,
	).Scan(&committedAmount)
	if err != nil {
		return 0, fmt.Errorf("commit reservation: query active reservations: %w", err)
	}

	if committedAmount == 0 {
		return 0, ErrActiveReservationNotFound
	}

	_, err = tx.Exec(ctx,
		`UPDATE user_balance_reservations
         SET status = 'committed', archived_at = NOW(), committed_at = NOW()
         WHERE user_id = $1 AND room_id = $2 AND status = 'active'`,
		userID,
		roomID,
	)
	if err != nil {
		return 0, fmt.Errorf("commit reservation: update reservation status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit reservation: commit transaction: %w", err)
	}

	return committedAmount, nil
}

// CommitReservationsWithWin finalizes active reservations and credits the user with a winning payout.
func CommitReservationsWithWin(
	ctx context.Context,
	db *pgxpool.Pool,
	userID int64,
	roomID int64,
	winAmount int64,
) (int64, error) {
	if winAmount < 0 {
		return 0, ErrInvalidAmount
	}

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin commit reservation with win transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// can be deleted
	var committedAmount int64
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM user_balance_reservations
         WHERE user_id = $1 AND room_id = $2 AND status = 'active'
         FOR UPDATE`,
		userID,
		roomID,
	).Scan(&committedAmount)
	if err != nil {
		return 0, fmt.Errorf("commit reservation with win: query active reservations: %w", err)
	}

	if committedAmount == 0 {
		return 0, ErrActiveReservationNotFound
	}

	_, err = tx.Exec(ctx,
		`UPDATE user_balance_reservations
         SET status = 'committed', archived_at = NOW(), committed_at = NOW()
         WHERE user_id = $1 AND room_id = $2 AND status = 'active'`,
		userID,
		roomID,
	)
	if err != nil {
		return 0, fmt.Errorf("commit reservation with win: update reservation status: %w", err)
	}

	if winAmount > 0 {
		_, err = tx.Exec(ctx,
			`UPDATE users SET balance = balance + $1 WHERE user_id = $2`,
			winAmount,
			userID,
		)
		if err != nil {
			return 0, fmt.Errorf("commit reservation with win: credit user balance: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit reservation with win: commit transaction: %w", err)
	}

	return committedAmount, nil
}

// user_id, room_id is not composite key
// reservation_type - вход в комнату или оплата boost