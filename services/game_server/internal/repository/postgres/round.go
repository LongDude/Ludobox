package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *txScope) CreateParticipant(ctx context.Context, userID, roundID int64, numberInRoom int) (int64, error) {
	var id int64
	err := s.tx.QueryRow(ctx, `
		INSERT INTO round_participants (user_id, rounds_id, number_in_room)
		VALUES ($1, $2, $3)
		RETURNING round_participants_id`,
		userID, roundID, numberInRoom,
	).Scan(&id)
	return id, err
}

func (s *txScope) GetParticipantUserID(ctx context.Context, participantID int64) (int64, error) {
	var uid int64
	err := s.tx.QueryRow(ctx, `SELECT user_id FROM round_participants WHERE round_participants_id = $1`, participantID).Scan(&uid)
	return uid, err
}

func (s *txScope) UpdateParticipantBoost(ctx context.Context, participantID int64, boost int) error {
	_, err := s.tx.Exec(ctx, `UPDATE round_participants SET boost = $1 WHERE round_participants_id = $2`, boost, participantID)
	return err
}

func (s *txScope) MarkParticipantExited(ctx context.Context, participantID int64) error {
	_, err := s.tx.Exec(ctx, `UPDATE round_participants SET exit_room_at = NOW() WHERE round_participants_id = $1`, participantID)
	return err
}

func (s *txScope) UpdateWinningMoney(ctx context.Context, participantID int64, amount int64) error {
	_, err := s.tx.Exec(ctx, `UPDATE round_participants SET winning_money = $1 WHERE round_participants_id = $2`, amount, participantID)
	return err
}

func (s *txScope) ArchiveRound(ctx context.Context, roundID int64) error {
	_, err := s.tx.Exec(ctx, `UPDATE rounds SET archived_at = NOW() WHERE rounds_id = $1`, roundID)
	return err
}

func (s *txScope) CreateRound(ctx context.Context, roomID int64) (int64, error) {
	var id int64
	err := s.tx.QueryRow(ctx, `
		INSERT INTO rounds (room_id, created_at)
		VALUES ($1, NOW())
		RETURNING rounds_id`,
		roomID,
	).Scan(&id)
	return id, err
}

func (s *txScope) UpdateRoundStatus(ctx context.Context, roundID int64, status string) error {
	_, err := s.tx.Exec(ctx, `UPDATE rounds SET status = $1 WHERE rounds_id = $2`, status, roundID)
	return err
}

func (s *txScope) GetActiveParticipantsCount(ctx context.Context, roundID int64) (int, error) {
	var count int
	err := s.tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM round_participants
		WHERE rounds_id = $1 AND exit_room_at IS NULL
	`, roundID).Scan(&count)
	return count, err
}

func (s *txScope) FindFreeNumberInRoom(ctx context.Context, roundID int64, capacity int) (int, error) {
	// Find the first free number from 1 to capacity
	for number := 1; number <= capacity; number++ {
		var exists bool
		err := s.tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM round_participants
				WHERE rounds_id = $1 AND number_in_room = $2 AND exit_room_at IS NULL
			)
		`, roundID, number).Scan(&exists)
		if err != nil {
			return 0, err
		}
		if !exists {
			return number, nil
		}
	}
	return 0, errors.New("no free spots in room")
}

func (s *txScope) GetRoundStatus(ctx context.Context, roundID int64) (string, error) {
	var status string
	err := s.tx.QueryRow(ctx, `
		SELECT status
		FROM rounds
		WHERE rounds_id = $1
	`, roundID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("round not found")
		}
		return "", err
	}
	return status, nil
}
