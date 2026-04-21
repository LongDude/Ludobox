package postgres

import (
	"context"
	"fmt"

	"game_server/internal/domain"
	"game_server/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type roomRepo struct {
	db *pgxpool.Pool
}

func NewRoomRepository(db *pgxpool.Pool) repository.RoomRepository {
	return &roomRepo{db: db}
}

func (r *roomRepo) InTransaction(ctx context.Context, fn func(ts repository.TransactionScope) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ts := &txScope{tx: tx}
	if err := fn(ts); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type txScope struct {
	tx pgx.Tx
}

// GetRoomsByServerID returns all rooms owned by a specific server
func (r *roomRepo) GetRoomsByServerID(ctx context.Context, serverID int64) ([]domain.Room, error) {
	rows, err := r.db.Query(ctx, `
		SELECT room_id, config_id, server_id, status, archived_at
		FROM rooms
		WHERE server_id = $1 AND archived_at IS NULL
	`, serverID)
	if err != nil {
		return nil, fmt.Errorf("query rooms: %w", err)
	}
	defer rows.Close()

	var rooms []domain.Room
	for rows.Next() {
		var room domain.Room
		if err := rows.Scan(&room.RoomID, &room.ConfigID, &room.ServerID, &room.Status, &room.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan room: %w", err)
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

// GetRoom returns detailed room information including config and active participants count
func (r *roomRepo) GetRoom(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	// Get room info
	var room domain.Room
	err := r.db.QueryRow(ctx, `
		SELECT room_id, config_id, server_id, status, archived_at
		FROM rooms
		WHERE room_id = $1
	`, roomID).Scan(&room.RoomID, &room.ConfigID, &room.ServerID, &room.Status, &room.ArchivedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("room not found")
		}
		return nil, fmt.Errorf("query room: %w", err)
	}

	// Get config info
	var config domain.RoomConfig
	err = r.db.QueryRow(ctx, `
		SELECT config_id, game_id, capacity, registration_price, is_boost, boost_price, 
		       boost_power, number_winners, winning_distribution, commission, time, min_users, archived_at
		FROM config
		WHERE config_id = $1
	`, room.ConfigID).Scan(&config.ConfigID, &config.GameID, &config.Capacity, &config.RegistrationPrice,
		&config.IsBoost, &config.BoostPrice, &config.BoostPower, &config.NumberWinners,
		&config.WinningDistribution, &config.Commission, &config.Time, &config.MinUsers, &config.ArchivedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("config not found")
		}
		return nil, fmt.Errorf("query config: %w", err)
	}

	// Get current round ID
	var currentRoundID *int64
	err = r.db.QueryRow(ctx, `
		SELECT rounds_id
		FROM rounds
		WHERE room_id = $1 AND archived_at IS NULL
		LIMIT 1
	`, roomID).Scan(&currentRoundID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("query current round: %w", err)
	}

	// Get active participants count
	var count int
	if currentRoundID != nil {
		err = r.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM round_participants
			WHERE rounds_id = $1 AND exit_room_at IS NULL
		`, *currentRoundID).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("query participants count: %w", err)
		}
	}

	return &domain.RoomInfo{
		Room:                    &room,
		Config:                  &config,
		CurrentRoundID:          currentRoundID,
		ActiveParticipantsCount: count,
	}, nil
}

// GetRoomConfig returns configuration for a room
func (r *roomRepo) GetRoomConfig(ctx context.Context, configID int64) (*domain.RoomConfig, error) {
	var config domain.RoomConfig
	err := r.db.QueryRow(ctx, `
		SELECT config_id, game_id, capacity, registration_price, is_boost, boost_price, 
		       boost_power, number_winners, winning_distribution, commission, time, min_users, archived_at
		FROM config
		WHERE config_id = $1
	`, configID).Scan(&config.ConfigID, &config.GameID, &config.Capacity, &config.RegistrationPrice,
		&config.IsBoost, &config.BoostPrice, &config.BoostPower, &config.NumberWinners,
		&config.WinningDistribution, &config.Commission, &config.Time, &config.MinUsers, &config.ArchivedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("config not found")
		}
		return nil, fmt.Errorf("query config: %w", err)
	}
	return &config, nil
}

// GetCurrentRoundByRoomID returns the ID of the current active (non-archived) round for a room
func (r *roomRepo) GetCurrentRoundByRoomID(ctx context.Context, roomID int64) (*int64, error) {
	var roundID int64
	err := r.db.QueryRow(ctx, `
		SELECT rounds_id
		FROM rounds
		WHERE room_id = $1 AND archived_at IS NULL
		LIMIT 1
	`, roomID).Scan(&roundID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No active round
		}
		return nil, fmt.Errorf("query current round: %w", err)
	}
	return &roundID, nil
}

// GetRoundInfo returns information about a round
func (r *roomRepo) GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error) {
	var round domain.Round
	err := r.db.QueryRow(ctx, `
		SELECT rounds_id, room_id, created_at, archived_at
		FROM rounds
		WHERE rounds_id = $1
	`, roundID).Scan(&round.RoundsID, &round.RoomID, &round.CreatedAt, &round.ArchivedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("round not found")
		}
		return nil, fmt.Errorf("query round: %w", err)
	}
	return &round, nil
}

// GetParticipantByID returns a single participant by ID
func (r *roomRepo) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	var p domain.RoundParticipant
	err := r.db.QueryRow(ctx, `
		SELECT round_participants_id, user_id, rounds_id, boost, winning_money, number_in_room, exit_room_at 
		FROM round_participants WHERE round_participants_id = $1`,
		participantID,
	).Scan(&p.RoundParticipantID, &p.UserID, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("participant not found")
		}
		return nil, fmt.Errorf("get participant: %w", err)
	}
	return &p, nil
}

// GetParticipantsByRoundID returns all active (non-exited) participants for a round
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
