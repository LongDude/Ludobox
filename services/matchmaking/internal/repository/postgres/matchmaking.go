package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"user_service/internal/domain"

	"github.com/jackc/pgx/v5"
)

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (ar *internalRepository) RecommendRooms(ctx context.Context, preferences domain.MatchmakingPreferences) (domain.ListResponse[domain.RoomRecommendation], error) {
	staleAfterSeconds := normalizeStaleAfterSeconds(preferences.StaleAfter)
	response := domain.ListResponse[domain.RoomRecommendation]{
		Items: make([]domain.RoomRecommendation, 0),
	}

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
			COUNT(*) OVER()::BIGINT AS total_count,
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
		LIMIT $10 OFFSET $11;
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
		preferences.Offset,
	)
	if err != nil {
		return response, fmt.Errorf("recommend rooms: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var recommendation domain.RoomRecommendation
		var totalCount int64
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
			&totalCount,
			&recommendation.Score,
		); scanErr != nil {
			return response, fmt.Errorf("scan recommendation: %w", scanErr)
		}

		response.Total = totalCount
		response.Items = append(response.Items, recommendation)
	}
	if err := rows.Err(); err != nil {
		return response, fmt.Errorf("iterate recommendations: %w", err)
	}

	return response, nil
}

func (ar *internalRepository) GetUserActiveRoom(ctx context.Context, userID int64) (*domain.RoomMembership, error) {
	membership, err := loadUserActiveRoom(ctx, ar.db, userID)
	if err != nil {
		return nil, err
	}
	return membership, nil
}

func (ar *internalRepository) JoinRoom(ctx context.Context, userID int64, roomID int64, staleAfter time.Duration) (*domain.RoomMembership, error) {
	recommendation, err := getJoinableRoom(ctx, ar.db, roomID, staleAfter)
	if err != nil {
		return nil, err
	}

	return ar.joinRoomViaGameServer(ctx, userID, recommendation)
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

func getJoinableRoom(ctx context.Context, db queryRower, roomID int64, staleAfter time.Duration) (*domain.RoomRecommendation, error) {
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
	`

	var recommendation domain.RoomRecommendation
	if err := db.QueryRow(ctx, query, roomID, staleAfterSeconds).Scan(
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

func normalizeStaleAfterSeconds(staleAfter time.Duration) int64 {
	staleAfterSeconds := int64(staleAfter / time.Second)
	if staleAfterSeconds <= 0 {
		return 1
	}
	return staleAfterSeconds
}

type gameServerJoinRoomResponse struct {
	ParticipantID  int64 `json:"participant_id"`
	RoundID        int64 `json:"round_id"`
	NumberInRoom   int   `json:"number_in_room"`
	CurrentPlayers int   `json:"current_players"`
}

type gameServerErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (ar *internalRepository) joinRoomViaGameServer(ctx context.Context, userID int64, recommendation *domain.RoomRecommendation) (*domain.RoomMembership, error) {
	if recommendation == nil {
		return nil, domain.ErrorRoomUnavailable
	}
	if recommendation.InstanceKey == "" {
		return nil, domain.ErrorGameServerUnavailable
	}

	url := fmt.Sprintf("http://%s:8080/api/rooms/%d/join", recommendation.InstanceKey, recommendation.RoomID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build join room request: %w", err)
	}
	request.Header.Set("X-Authenticated-User", strconv.FormatInt(userID, 10))

	response, err := ar.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: join room request failed", domain.ErrorGameServerUnavailable)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read join room response: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, mapJoinRoomError(response.StatusCode, body)
	}

	var joinResponse gameServerJoinRoomResponse
	if err := json.Unmarshal(body, &joinResponse); err != nil {
		return nil, fmt.Errorf("decode join room response: %w", err)
	}

	recommendationCopy := *recommendation
	recommendationCopy.CurrentPlayers = int32(joinResponse.CurrentPlayers)

	return &domain.RoomMembership{
		RoomRecommendation: recommendationCopy,
		RoundID:            joinResponse.RoundID,
		RoundParticipantID: joinResponse.ParticipantID,
		SeatNumber:         int32(joinResponse.NumberInRoom),
	}, nil
}

func mapJoinRoomError(statusCode int, body []byte) error {
	var errorResponse gameServerErrorResponse
	_ = json.Unmarshal(body, &errorResponse)

	switch errorResponse.Code {
	case "ROOM_FULL":
		return domain.ErrorRoomFull
	case "ROOM_NOT_FOUND", "ROUND_NOT_JOINABLE", "GAME_STARTED":
		return domain.ErrorRoomUnavailable
	case "WRONG_GAME_SERVER":
		return domain.ErrorGameServerUnavailable
	}

	switch statusCode {
	case http.StatusNotFound, http.StatusConflict:
		return domain.ErrorRoomUnavailable
	case http.StatusPaymentRequired:
		return fmt.Errorf("join room rejected: %s", firstNonEmpty(errorResponse.Error, errorResponse.Message, "insufficient balance"))
	default:
		if statusCode >= http.StatusInternalServerError {
			return domain.ErrorGameServerUnavailable
		}
		return fmt.Errorf("join room failed with status %d", statusCode)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
