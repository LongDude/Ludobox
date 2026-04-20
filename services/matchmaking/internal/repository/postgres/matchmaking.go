package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"user_service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (ar *internalRepository) RecommendRooms(ctx context.Context, preferences domain.MatchmakingPreferences) ([]domain.RoomRecommendation, error) {
	staleAfterSeconds := normalizeStaleAfterSeconds(preferences.StaleAfter)

	var (
		gameID               any
		minRegistrationPrice any
		maxRegistrationPrice any
		minCapacity          any
		maxCapacity          any
		isBoost              any
		minBoostPower        any
	)

	if preferences.GameID != nil {
		gameID = *preferences.GameID
	}
	if preferences.MinRegistrationPrice != nil {
		minRegistrationPrice = *preferences.MinRegistrationPrice
	}
	if preferences.MaxRegistrationPrice != nil {
		maxRegistrationPrice = *preferences.MaxRegistrationPrice
	}
	if preferences.MinCapacity != nil {
		minCapacity = *preferences.MinCapacity
	}
	if preferences.MaxCapacity != nil {
		maxCapacity = *preferences.MaxCapacity
	}
	if preferences.IsBoost != nil {
		isBoost = *preferences.IsBoost
	}
	if preferences.MinBoostPower != nil {
		minBoostPower = *preferences.MinBoostPower
	}

	const query = `
		WITH user_history AS (
			SELECT
				COUNT(*)::BIGINT AS games_played,
				COALESCE(AVG(c.registration_price)::NUMERIC, 0) AS avg_registration_price,
				COALESCE(AVG(c.capacity)::NUMERIC, 0) AS avg_capacity,
				(
					SELECT c2.game_id
					FROM round_participants rp2
					INNER JOIN rounds rd2 ON rd2.rounds_id = rp2.rounds_id
					INNER JOIN rooms r2 ON r2.room_id = rd2.room_id
					INNER JOIN config c2 ON c2.config_id = r2.config_id
					WHERE rp2.user_id = $1
					GROUP BY c2.game_id
					ORDER BY COUNT(*) DESC, MAX(rd2.created_at) DESC
					LIMIT 1
				) AS preferred_game_id
			FROM round_participants rp
			INNER JOIN rounds rd ON rd.rounds_id = rp.rounds_id
			INNER JOIN rooms r ON r.room_id = rd.room_id
			INNER JOIN config c ON c.config_id = r.config_id
			WHERE rp.user_id = $1
		),
		active_players AS (
			SELECT
				rd.room_id,
				COUNT(rp.round_participants_id) FILTER (WHERE rp.exit_room_at IS NULL)::INT AS current_players
			FROM rounds rd
			LEFT JOIN round_participants rp ON rp.rounds_id = rd.rounds_id
			WHERE rd.archived_at IS NULL
			GROUP BY rd.room_id
		)
		SELECT
			r.room_id,
			r.config_id,
			r.server_id,
			c.game_id,
			c.registration_price,
			c.capacity,
			c.min_users,
			c.is_boost,
			c.boost_power,
			COALESCE(ap.current_players, 0) AS current_players,
			gs.instance_key,
			gs.redis_host,
			(
				CASE
					WHEN uh.preferred_game_id IS NOT NULL AND c.game_id = uh.preferred_game_id THEN 40
					ELSE 0
				END
				+
				CASE
					WHEN uh.games_played = 0 THEN 12
					WHEN uh.avg_registration_price <= 0 THEN 8
					ELSE GREATEST(
						0,
						22 - (ABS(c.registration_price::NUMERIC - uh.avg_registration_price) / GREATEST(uh.avg_registration_price, 1)) * 22
					)
				END
				+
				CASE
					WHEN uh.games_played = 0 THEN 8
					WHEN uh.avg_capacity <= 0 THEN 5
					ELSE GREATEST(
						0,
						18 - (ABS(c.capacity::NUMERIC - uh.avg_capacity) / GREATEST(uh.avg_capacity, 1)) * 18
					)
				END
				+
				CASE
					WHEN c.min_users > 0 THEN LEAST(20, (COALESCE(ap.current_players, 0)::NUMERIC / c.min_users::NUMERIC) * 20)
					ELSE 0
				END
			)::DOUBLE PRECISION AS score
		FROM rooms r
		INNER JOIN config c ON c.config_id = r.config_id AND c.archived_at IS NULL
		INNER JOIN game_servers gs ON gs.server_id = r.server_id
		LEFT JOIN active_players ap ON ap.room_id = r.room_id
		CROSS JOIN user_history uh
		WHERE r.archived_at IS NULL
		  AND r.status = 'open'
		  AND gs.status = 'up'
		  AND gs.archived_at IS NULL
		  AND gs.last_heartbeat_at >= NOW() - make_interval(secs => $9)
		  AND COALESCE(ap.current_players, 0) < c.capacity
		  AND ($2::BIGINT IS NULL OR c.game_id = $2)
		  AND ($3::BIGINT IS NULL OR c.registration_price >= $3)
		  AND ($4::BIGINT IS NULL OR c.registration_price <= $4)
		  AND ($5::INT IS NULL OR c.capacity >= $5)
		  AND ($6::INT IS NULL OR c.capacity <= $6)
		  AND ($7::BOOLEAN IS NULL OR c.is_boost = $7)
		  AND ($8::INT IS NULL OR c.boost_power >= $8)
		ORDER BY score DESC, COALESCE(ap.current_players, 0) DESC, r.room_id ASC
		LIMIT $10;
	`

	rows, err := ar.db.Query(
		ctx,
		query,
		preferences.UserID,
		gameID,
		minRegistrationPrice,
		maxRegistrationPrice,
		minCapacity,
		maxCapacity,
		isBoost,
		minBoostPower,
		staleAfterSeconds,
		preferences.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recommend rooms: %w", err)
	}
	defer rows.Close()

	recommendations := make([]domain.RoomRecommendation, 0, preferences.Limit)
	for rows.Next() {
		var recommendation domain.RoomRecommendation
		if scanErr := rows.Scan(
			&recommendation.RoomID,
			&recommendation.ConfigID,
			&recommendation.ServerID,
			&recommendation.GameID,
			&recommendation.RegistrationPrice,
			&recommendation.Capacity,
			&recommendation.MinUsers,
			&recommendation.IsBoost,
			&recommendation.BoostPower,
			&recommendation.CurrentPlayers,
			&recommendation.InstanceKey,
			&recommendation.RedisHost,
			&recommendation.Score,
		); scanErr != nil {
			return nil, fmt.Errorf("scan recommendation: %w", scanErr)
		}

		recommendations = append(recommendations, recommendation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recommendations: %w", err)
	}

	return recommendations, nil
}

func (ar *internalRepository) GetUserActiveRoom(ctx context.Context, userID int64) (*domain.RoomMembership, error) {
	membership, err := loadUserActiveRoom(ctx, ar.db, userID)
	if err != nil {
		return nil, err
	}
	return membership, nil
}

func (ar *internalRepository) JoinRoom(ctx context.Context, userID int64, roomID int64, staleAfter time.Duration) (*domain.RoomMembership, error) {
	tx, err := ar.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin join room tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	currentMembership, err := loadUserActiveRoom(ctx, tx, userID)
	if err == nil {
		if currentMembership.RoomID == roomID {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return nil, fmt.Errorf("commit join room tx: %w", commitErr)
			}
			return currentMembership, nil
		}
		return nil, domain.ErrorUserAlreadyInRoom
	}
	if !errors.Is(err, domain.ErrorActiveRoomNotFound) {
		return nil, fmt.Errorf("check user active room: %w", err)
	}

	recommendation, err := lockJoinableRoom(ctx, tx, roomID, staleAfter)
	if err != nil {
		return nil, err
	}

	roundID, currentPlayers, err := ensureActiveRound(ctx, tx, roomID)
	if err != nil {
		return nil, err
	}
	if currentPlayers >= recommendation.Capacity {
		return nil, domain.ErrorRoomFull
	}

	seatNumber := currentPlayers + 1

	const insertParticipantQuery = `
		INSERT INTO round_participants (user_id, rounds_id, number_in_room)
		VALUES ($1, $2, $3)
		RETURNING round_participants_id;
	`

	var roundParticipantID int64
	if err := tx.QueryRow(ctx, insertParticipantQuery, userID, roundID, seatNumber).Scan(&roundParticipantID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23503" {
				return nil, domain.ErrorUserNotFound
			}
		}
		return nil, fmt.Errorf("insert round participant: %w", err)
	}

	recommendation.CurrentPlayers = seatNumber
	membership := &domain.RoomMembership{
		RoomRecommendation: *recommendation,
		RoundID:            roundID,
		RoundParticipantID: roundParticipantID,
		SeatNumber:         seatNumber,
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit join room tx: %w", err)
	}

	return membership, nil
}

func loadUserActiveRoom(ctx context.Context, db queryRower, userID int64) (*domain.RoomMembership, error) {
	const query = `
		SELECT
			r.room_id,
			r.config_id,
			r.server_id,
			c.game_id,
			c.registration_price,
			c.capacity,
			c.min_users,
			c.is_boost,
			c.boost_power,
			COALESCE(active_players.current_players, 0) AS current_players,
			gs.instance_key,
			gs.redis_host,
			rd.rounds_id,
			rp.round_participants_id,
			rp.number_in_room
		FROM round_participants rp
		INNER JOIN rounds rd ON rd.rounds_id = rp.rounds_id
		INNER JOIN rooms r ON r.room_id = rd.room_id
		INNER JOIN config c ON c.config_id = r.config_id
		INNER JOIN game_servers gs ON gs.server_id = r.server_id
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::INT AS current_players
			FROM round_participants rp2
			WHERE rp2.rounds_id = rd.rounds_id
			  AND rp2.exit_room_at IS NULL
		) active_players ON TRUE
		WHERE rp.user_id = $1
		  AND rp.exit_room_at IS NULL
		  AND rd.archived_at IS NULL
		  AND r.archived_at IS NULL
		  AND r.status = 'open'
		  AND c.archived_at IS NULL
		ORDER BY rd.created_at DESC, rp.round_participants_id DESC
		LIMIT 1;
	`

	var membership domain.RoomMembership
	if err := db.QueryRow(ctx, query, userID).Scan(
		&membership.RoomID,
		&membership.ConfigID,
		&membership.ServerID,
		&membership.GameID,
		&membership.RegistrationPrice,
		&membership.Capacity,
		&membership.MinUsers,
		&membership.IsBoost,
		&membership.BoostPower,
		&membership.CurrentPlayers,
		&membership.InstanceKey,
		&membership.RedisHost,
		&membership.RoundID,
		&membership.RoundParticipantID,
		&membership.SeatNumber,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrorActiveRoomNotFound
		}
		return nil, fmt.Errorf("load active room: %w", err)
	}

	membership.Score = 100
	return &membership, nil
}

func lockJoinableRoom(ctx context.Context, tx pgx.Tx, roomID int64, staleAfter time.Duration) (*domain.RoomRecommendation, error) {
	staleAfterSeconds := normalizeStaleAfterSeconds(staleAfter)

	const query = `
		SELECT
			r.room_id,
			r.config_id,
			r.server_id,
			c.game_id,
			c.registration_price,
			c.capacity,
			c.min_users,
			c.is_boost,
			c.boost_power,
			gs.instance_key,
			gs.redis_host
		FROM rooms r
		INNER JOIN config c ON c.config_id = r.config_id
		INNER JOIN game_servers gs ON gs.server_id = r.server_id
		WHERE r.room_id = $1
		  AND r.archived_at IS NULL
		  AND r.status = 'open'
		  AND c.archived_at IS NULL
		  AND gs.status = 'up'
		  AND gs.archived_at IS NULL
		  AND gs.last_heartbeat_at >= NOW() - make_interval(secs => $2)
		FOR UPDATE OF r;
	`

	var recommendation domain.RoomRecommendation
	if err := tx.QueryRow(ctx, query, roomID, staleAfterSeconds).Scan(
		&recommendation.RoomID,
		&recommendation.ConfigID,
		&recommendation.ServerID,
		&recommendation.GameID,
		&recommendation.RegistrationPrice,
		&recommendation.Capacity,
		&recommendation.MinUsers,
		&recommendation.IsBoost,
		&recommendation.BoostPower,
		&recommendation.InstanceKey,
		&recommendation.RedisHost,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrorRoomUnavailable
		}
		return nil, fmt.Errorf("lock room for join: %w", err)
	}

	return &recommendation, nil
}

func ensureActiveRound(ctx context.Context, tx pgx.Tx, roomID int64) (int64, int32, error) {
	const selectRoundQuery = `
		SELECT rounds_id
		FROM rounds
		WHERE room_id = $1
		  AND archived_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE;
	`

	var roundID int64
	if err := tx.QueryRow(ctx, selectRoundQuery, roomID).Scan(&roundID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, fmt.Errorf("select active round: %w", err)
		}

		const createRoundQuery = `
			INSERT INTO rounds (room_id)
			VALUES ($1)
			RETURNING rounds_id;
		`
		if createErr := tx.QueryRow(ctx, createRoundQuery, roomID).Scan(&roundID); createErr != nil {
			return 0, 0, fmt.Errorf("create active round: %w", createErr)
		}
	}

	const countPlayersQuery = `
		SELECT COUNT(*)::INT
		FROM round_participants
		WHERE rounds_id = $1
		  AND exit_room_at IS NULL;
	`

	var currentPlayers int32
	if err := tx.QueryRow(ctx, countPlayersQuery, roundID).Scan(&currentPlayers); err != nil {
		return 0, 0, fmt.Errorf("count active players: %w", err)
	}

	return roundID, currentPlayers, nil
}

func normalizeStaleAfterSeconds(staleAfter time.Duration) int64 {
	staleAfterSeconds := int64(staleAfter / time.Second)
	if staleAfterSeconds <= 0 {
		return 1
	}
	return staleAfterSeconds
}
