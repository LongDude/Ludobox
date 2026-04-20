package postgres

import (
	"context"
	"errors"
	"fmt"

	"game_server/internal/domain"

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

// --- Read-only методы (используют pool, а не tx) ---
func (r *roomRepo) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	var p domain.RoundParticipant
	err := r.db.QueryRow(ctx, `
		SELECT round_participants_id, user_id, rounds_id, boost, winning_money, number_in_room, exit_room_at 
		FROM round_participants WHERE round_participants_id = $1`,
		participantID,
	).Scan(&p.RoundParticipantID, &p.UserID, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("participant not found")
		}
		return nil, fmt.Errorf("get participant: %w", err)
	}
	return &p, nil
}

func (r *roomRepo) GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT round_participants_id, user_id, rounds_id, boost, winning_money, number_in_room, exit_room_at 
		FROM round_participants WHERE rounds_id = $1 AND exit_room_at IS NULL`,
		roundID,
	)
	if err != nil {
		return nil, fmt.Errorf("get participants: %w", err)
	}
	defer rows.Close()

	var participants []domain.RoundParticipant
	for rows.Next() {
		var p domain.RoundParticipant
		if err := rows.Scan(&p.RoundParticipantID, &p.UserID, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}
