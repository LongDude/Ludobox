package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"game_server/internal/domain"
	"game_server/internal/repository"

	"github.com/jackc/pgx/v5"
)

func (s *txScope) GetRoomForUpdate(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	var room domain.Room
	err := s.tx.QueryRow(ctx, `
		SELECT room_id, config_id, server_id, status, current_players, archived_at
		FROM rooms
		WHERE room_id = $1
		  AND archived_at IS NULL
		FOR UPDATE
	`, roomID).Scan(&room.RoomID, &room.ConfigID, &room.ServerID, &room.Status, &room.CurrentPlayers, &room.ArchivedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room for update: %w", err)
	}

	var config domain.RoomConfig
	err = s.tx.QueryRow(ctx, `
		SELECT c.config_id, c.game_id, g.name_game, c.capacity, c.registration_price, c.is_boost, c.boost_price,
		       c.boost_power, c.number_winners, c.winning_distribution, c.commission, c.time, c.round_time, c.next_round_delay, c.min_users, c.archived_at
		FROM config c
		INNER JOIN games g ON g.game_id = c.game_id
		WHERE c.config_id = $1
	`, room.ConfigID).Scan(&config.ConfigID, &config.GameID, &config.GameName, &config.Capacity, &config.RegistrationPrice,
		&config.IsBoost, &config.BoostPrice, &config.BoostPower, &config.NumberWinners,
		&config.WinningDistribution, &config.Commission, &config.Time, &config.RoundTime, &config.NextRoundDelay, &config.MinUsers, &config.ArchivedAt)
	if err != nil {
		return nil, fmt.Errorf("get room config for update: %w", err)
	}

	var currentRoundID *int64
	var currentRoundStatus *string
	err = s.tx.QueryRow(ctx, `
		SELECT rounds_id, status
		FROM rounds
		WHERE room_id = $1 AND archived_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE
	`, roomID).Scan(&currentRoundID, &currentRoundStatus)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get current round for update: %w", err)
	}

	activeParticipantsCount := 0
	if currentRoundID != nil {
		activeParticipantsCount, err = s.GetActiveParticipantsCount(ctx, *currentRoundID)
		if err != nil {
			return nil, fmt.Errorf("get active participants count: %w", err)
		}
	}

	return &domain.RoomInfo{
		Room:                    &room,
		Config:                  &config,
		CurrentRoundID:          currentRoundID,
		CurrentRoundStatus:      currentRoundStatus,
		ActiveParticipantsCount: activeParticipantsCount,
	}, nil
}

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

func (s *txScope) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	var p domain.RoundParticipant
	var nickname sql.NullString
	var rating sql.NullInt64
	err := s.tx.QueryRow(ctx, `
		SELECT rp.round_participants_id, rp.user_id, u.nickname, u.rating, rp.rounds_id, rp.boost, rp.winning_money, rp.number_in_room, rp.exit_room_at
		FROM round_participants rp
		INNER JOIN users u ON u.user_id = rp.user_id
		WHERE rp.round_participants_id = $1
		FOR UPDATE OF rp
	`, participantID).Scan(&p.RoundParticipantID, &p.UserID, &nickname, &rating, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrParticipantNotFound
		}
		return nil, fmt.Errorf("get participant by id: %w", err)
	}
	if nickname.Valid && nickname.String != "" {
		p.NickName = &nickname.String
	}
	if rating.Valid {
		p.Rating = &rating.Int64
	}
	return &p, nil
}

func (s *txScope) GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	rows, err := s.tx.Query(ctx, `
		SELECT rp.round_participants_id, rp.user_id, u.nickname, u.rating, rp.rounds_id, rp.boost, rp.winning_money, rp.number_in_room, rp.exit_room_at
		FROM round_participants rp
		INNER JOIN users u ON u.user_id = rp.user_id
		WHERE rp.rounds_id = $1 AND rp.exit_room_at IS NULL
		ORDER BY rp.number_in_room
	`, roundID)
	if err != nil {
		return nil, fmt.Errorf("get participants by round: %w", err)
	}
	defer rows.Close()

	var participants []domain.RoundParticipant
	for rows.Next() {
		var p domain.RoundParticipant
		var nickname sql.NullString
		var rating sql.NullInt64
		if err := rows.Scan(&p.RoundParticipantID, &p.UserID, &nickname, &rating, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		if nickname.Valid && nickname.String != "" {
			p.NickName = &nickname.String
		}
		if rating.Valid {
			p.Rating = &rating.Int64
		}
		participants = append(participants, p)
	}

	return participants, rows.Err()
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

func (s *txScope) GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error) {
	var round domain.Round
	err := s.tx.QueryRow(ctx, `
		SELECT rounds_id, room_id, status, created_at, archived_at
		FROM rounds
		WHERE rounds_id = $1
		FOR UPDATE
	`, roundID).Scan(&round.RoundsID, &round.RoomID, &round.Status, &round.CreatedAt, &round.ArchivedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrRoundArchived
		}
		return nil, fmt.Errorf("get round info: %w", err)
	}
	return &round, nil
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

func (s *txScope) CountUserActiveParticipants(ctx context.Context, roundID, userID int64) (int, error) {
	var count int
	err := s.tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM round_participants
		WHERE rounds_id = $1
		  AND user_id = $2
		  AND exit_room_at IS NULL
	`, roundID, userID).Scan(&count)
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

func (s *txScope) IsSeatOccupied(ctx context.Context, roundID int64, numberInRoom int) (bool, error) {
	var exists bool
	err := s.tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM round_participants
			WHERE rounds_id = $1
			  AND number_in_room = $2
			  AND exit_room_at IS NULL
		)
	`, roundID, numberInRoom).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check seat occupied: %w", err)
	}
	return exists, nil
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
			return "", repository.ErrRoundArchived
		}
		return "", err
	}
	return status, nil
}

func (s *txScope) SetRoomCurrentPlayers(ctx context.Context, roomID int64, currentPlayers int) error {
	_, err := s.tx.Exec(ctx, `
		UPDATE rooms
		SET current_players = $2
		WHERE room_id = $1
	`, roomID, currentPlayers)
	if err != nil {
		return fmt.Errorf("set room current players: %w", err)
	}
	return nil
}
