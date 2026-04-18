package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"user_service/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (ar *internalRepository) ResolveRoomOwner(ctx context.Context, roomID int64) (*domain.GameServer, error) {
	query := `
		SELECT
			gs.server_id,
			gs.instance_key,
			gs.redis_host,
			gs.status,
			gs.last_heartbeat_at,
			gs.archived_at
		FROM rooms AS r
		INNER JOIN game_servers AS gs ON gs.server_id = r.server_id
		WHERE r.room_id = $1
		  AND r.archived_at IS NULL;
	`

	var gameServer domain.GameServer
	if err := ar.db.QueryRow(ctx, query, roomID).Scan(
		&gameServer.ServerID,
		&gameServer.InstanceKey,
		&gameServer.RedisHost,
		&gameServer.Status,
		&gameServer.LastHeartbeatAt,
		&gameServer.ArchivedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrorRoomNotFound
		}
		return nil, fmt.Errorf("resolve room owner: %w", err)
	}

	return &gameServer, nil
}

func (ar *internalRepository) SelectAvailableGameServer(ctx context.Context, staleAfter time.Duration) (*domain.GameServer, error) {
	staleAfterSeconds := int64(staleAfter / time.Second)
	if staleAfterSeconds <= 0 {
		staleAfterSeconds = 1
	}

	query := `
		SELECT
			gs.server_id,
			gs.instance_key,
			gs.redis_host,
			gs.status,
			gs.last_heartbeat_at,
			gs.archived_at,
			COUNT(r.room_id) AS active_rooms
		FROM game_servers AS gs
		LEFT JOIN rooms AS r
			ON r.server_id = gs.server_id
		   AND r.archived_at IS NULL
		   AND r.status IN ('open', 'in_game')
		WHERE gs.status = 'up'
		  AND gs.archived_at IS NULL
		  AND gs.last_heartbeat_at >= NOW() - make_interval(secs => $1)
		GROUP BY gs.server_id
		ORDER BY active_rooms ASC, gs.server_id ASC
		LIMIT 1;
	`

	var gameServer domain.GameServer
	if err := ar.db.QueryRow(ctx, query, staleAfterSeconds).Scan(
		&gameServer.ServerID,
		&gameServer.InstanceKey,
		&gameServer.RedisHost,
		&gameServer.Status,
		&gameServer.LastHeartbeatAt,
		&gameServer.ArchivedAt,
		&gameServer.ActiveRooms,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrorNoActiveGameServers
		}
		return nil, fmt.Errorf("select available game server: %w", err)
	}

	return &gameServer, nil
}
