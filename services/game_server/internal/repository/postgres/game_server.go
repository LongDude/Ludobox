package postgres

import (
	"context"
	"fmt"
	"game_server/internal/domain"
	"game_server/internal/repository"
)

func (ar *internalRepository) RegisterGameServer(ctx context.Context, params repository.GameServerRegistrationParams) (*domain.GameServer, error) {
	query := `
		INSERT INTO game_servers (
			instance_key,
			redis_host,
			status,
			started_at,
			last_heartbeat_at,
			archived_at
		)
		VALUES ($1, $2, 'up', NOW(), NOW(), NULL)
		ON CONFLICT (instance_key)
		DO UPDATE SET
			redis_host = EXCLUDED.redis_host,
			status = 'up',
			last_heartbeat_at = NOW(),
			archived_at = NULL
		RETURNING server_id, instance_key, redis_host, status, last_heartbeat_at, archived_at;
	`

	var gameServer domain.GameServer
	if err := ar.db.QueryRow(ctx, query, params.InstanceKey, params.RedisHost).Scan(
		&gameServer.ServerID,
		&gameServer.InstanceKey,
		&gameServer.RedisHost,
		&gameServer.Status,
		&gameServer.LastHeartbeatAt,
		&gameServer.ArchivedAt,
	); err != nil {
		return nil, fmt.Errorf("register game server: %w", err)
	}

	return &gameServer, nil
}

func (ar *internalRepository) HeartbeatGameServer(ctx context.Context, serverID int64) error {
	query := `
		UPDATE game_servers
		SET status = 'up',
			last_heartbeat_at = NOW(),
			archived_at = NULL
		WHERE server_id = $1;
	`

	if _, err := ar.db.Exec(ctx, query, serverID); err != nil {
		return fmt.Errorf("update game server heartbeat: %w", err)
	}

	return nil
}

func (ar *internalRepository) DeactivateGameServer(ctx context.Context, serverID int64) error {
	query := `
		UPDATE game_servers
		SET status = 'down',
			archived_at = NOW()
		WHERE server_id = $1;
	`

	if _, err := ar.db.Exec(ctx, query, serverID); err != nil {
		return fmt.Errorf("deactivate game server: %w", err)
	}

	return nil
}
