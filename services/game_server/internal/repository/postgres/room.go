package postgres

import (
	"context"
	"database/sql"
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
		SELECT room_id, config_id, server_id, status, current_players, archived_at
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
		if err := rows.Scan(&room.RoomID, &room.ConfigID, &room.ServerID, &room.Status, &room.CurrentPlayers, &room.ArchivedAt); err != nil {
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
		SELECT room_id, config_id, server_id, status, current_players, archived_at
		FROM rooms
		WHERE room_id = $1
		  AND archived_at IS NULL
	`, roomID).Scan(&room.RoomID, &room.ConfigID, &room.ServerID, &room.Status, &room.CurrentPlayers, &room.ArchivedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query room: %w", err)
	}

	// Get config info
	var config domain.RoomConfig
	err = r.db.QueryRow(ctx, `
		SELECT c.config_id, c.game_id, g.name_game, c.capacity, c.registration_price, c.is_boost, c.boost_price,
		       c.boost_power, c.number_winners, c.winning_distribution, c.commission, c.time, c.round_time, c.next_round_delay, c.min_users, c.archived_at
		FROM config c
		INNER JOIN games g ON g.game_id = c.game_id
		WHERE c.config_id = $1
	`, room.ConfigID).Scan(&config.ConfigID, &config.GameID, &config.GameName, &config.Capacity, &config.RegistrationPrice,
		&config.IsBoost, &config.BoostPrice, &config.BoostPower, &config.NumberWinners,
		&config.WinningDistribution, &config.Commission, &config.Time, &config.RoundTime, &config.NextRoundDelay, &config.MinUsers, &config.ArchivedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("config not found")
		}
		return nil, fmt.Errorf("query config: %w", err)
	}

	// Get current round ID
	var currentRoundID *int64
	var currentRoundStatus *string
	err = r.db.QueryRow(ctx, `
		SELECT rounds_id, status
		FROM rounds
		WHERE room_id = $1 AND archived_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`, roomID).Scan(&currentRoundID, &currentRoundStatus)
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
		CurrentRoundStatus:      currentRoundStatus,
		ActiveParticipantsCount: count,
	}, nil
}

// GetRoomConfig returns configuration for a room
func (r *roomRepo) GetRoomConfig(ctx context.Context, configID int64) (*domain.RoomConfig, error) {
	var config domain.RoomConfig
	err := r.db.QueryRow(ctx, `
		SELECT c.config_id, c.game_id, g.name_game, c.capacity, c.registration_price, c.is_boost, c.boost_price,
		       c.boost_power, c.number_winners, c.winning_distribution, c.commission, c.time, c.round_time, c.next_round_delay, c.min_users, c.archived_at
		FROM config c
		INNER JOIN games g ON g.game_id = c.game_id
		WHERE c.config_id = $1
	`, configID).Scan(&config.ConfigID, &config.GameID, &config.GameName, &config.Capacity, &config.RegistrationPrice,
		&config.IsBoost, &config.BoostPrice, &config.BoostPower, &config.NumberWinners,
		&config.WinningDistribution, &config.Commission, &config.Time, &config.RoundTime, &config.NextRoundDelay, &config.MinUsers, &config.ArchivedAt)
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
		ORDER BY created_at DESC
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
		SELECT rounds_id, room_id, status, created_at, archived_at
		FROM rounds
		WHERE rounds_id = $1
	`, roundID).Scan(&round.RoundsID, &round.RoomID, &round.Status, &round.CreatedAt, &round.ArchivedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repository.ErrRoundArchived
		}
		return nil, fmt.Errorf("query round: %w", err)
	}
	return &round, nil
}

// GetParticipantByID returns a single participant by ID
func (r *roomRepo) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	var p domain.RoundParticipant
	var nickname sql.NullString
	err := r.db.QueryRow(ctx, `
		SELECT rp.round_participants_id, rp.user_id, u.nickname, rp.rounds_id, rp.boost, rp.winning_money, rp.number_in_room, rp.exit_room_at
		FROM round_participants rp
		INNER JOIN users u ON u.user_id = rp.user_id
		WHERE rp.round_participants_id = $1`,
		participantID,
	).Scan(&p.RoundParticipantID, &p.UserID, &nickname, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repository.ErrParticipantNotFound
		}
		return nil, fmt.Errorf("get participant: %w", err)
	}
	if nickname.Valid && nickname.String != "" {
		p.NickName = &nickname.String
	}
	return &p, nil
}

// GetParticipantsByRoundID returns all active (non-exited) participants for a round
func (r *roomRepo) GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rp.round_participants_id, rp.user_id, u.nickname, rp.rounds_id, rp.boost, rp.winning_money, rp.number_in_room, rp.exit_room_at
		FROM round_participants rp
		INNER JOIN users u ON u.user_id = rp.user_id
		WHERE rp.rounds_id = $1 AND rp.exit_room_at IS NULL
		ORDER BY rp.number_in_room`,
		roundID,
	)
	if err != nil {
		return nil, fmt.Errorf("get participants: %w", err)
	}
	defer rows.Close()

	var participants []domain.RoundParticipant
	for rows.Next() {
		var p domain.RoundParticipant
		var nickname sql.NullString
		if err := rows.Scan(&p.RoundParticipantID, &p.UserID, &nickname, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt); err != nil {
			return nil, err
		}
		if nickname.Valid && nickname.String != "" {
			p.NickName = &nickname.String
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (r *roomRepo) GetActiveParticipantsByRoomAndUser(ctx context.Context, roomID, userID int64) ([]domain.RoundParticipant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rp.round_participants_id, rp.user_id, u.nickname, rp.rounds_id, rp.boost, rp.winning_money, rp.number_in_room, rp.exit_room_at
		FROM round_participants rp
		INNER JOIN rounds r ON r.rounds_id = rp.rounds_id
		INNER JOIN users u ON u.user_id = rp.user_id
		WHERE r.room_id = $1
		  AND r.archived_at IS NULL
		  AND rp.user_id = $2
		  AND rp.exit_room_at IS NULL
		ORDER BY rp.number_in_room
	`, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("get user participants by room: %w", err)
	}
	defer rows.Close()

	var participants []domain.RoundParticipant
	for rows.Next() {
		var p domain.RoundParticipant
		var nickname sql.NullString
		if err := rows.Scan(&p.RoundParticipantID, &p.UserID, &nickname, &p.RoundsID, &p.Boost, &p.WinningMoney, &p.NumberInRoom, &p.ExitRoomAt); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		if nickname.Valid && nickname.String != "" {
			p.NickName = &nickname.String
		}
		participants = append(participants, p)
	}

	return participants, rows.Err()
}
