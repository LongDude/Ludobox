package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"user_service/internal/domain"
	"user_service/internal/repository"
)

var gameServerSortableColumns = map[string]string{
	"server_id":         "server_id",
	"instance_key":      "instance_key",
	"redis_host":        "redis_host",
	"status":            "status",
	"started_at":        "started_at",
	"last_heartbeat_at": "last_heartbeat_at",
	"archived_at":       "archived_at",
}

func (r *gameServerRepository) GetServers(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.GameServer], error) {
	response := domain.ListResponse[domain.GameServer]{
		Items: make([]domain.GameServer, 0),
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	orderBy, err := buildGameServerOrderBy(params.Sort)
	if err != nil {
		return response, err
	}

	const countQuery = `SELECT COUNT(*) FROM game_servers`
	if err := r.db.QueryRow(ctx, countQuery).Scan(&response.Total); err != nil {
		return response, fmt.Errorf("count game servers: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			server_id,
			instance_key,
			redis_host,
			status,
			started_at,
			last_heartbeat_at,
			archived_at
		FROM game_servers
		ORDER BY %s
		LIMIT $1 OFFSET $2
	`, orderBy)

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return response, fmt.Errorf("list game servers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		server, scanErr := scanGameServer(rows)
		if scanErr != nil {
			return response, fmt.Errorf("scan game server: %w", scanErr)
		}
		response.Items = append(response.Items, *server)
	}
	if err := rows.Err(); err != nil {
		return response, fmt.Errorf("iterate game servers: %w", err)
	}

	return response, nil
}

func buildGameServerOrderBy(sort *domain.Sort) (string, error) {
	if sort == nil {
		return "server_id DESC", nil
	}

	column, ok := gameServerSortableColumns[strings.ToLower(strings.TrimSpace(sort.Field))]
	if !ok {
		return "", fmt.Errorf("%w: unsupported sort field %q", repository.ErrorInvalidListParams, sort.Field)
	}

	direction := strings.ToUpper(strings.TrimSpace(sort.Direction))
	if direction == "" {
		direction = "ASC"
	}
	if direction != "ASC" && direction != "DESC" {
		return "", fmt.Errorf("%w: unsupported sort direction %q", repository.ErrorInvalidListParams, sort.Direction)
	}

	return column + " " + direction, nil
}

func scanGameServer(row rowScanner) (*domain.GameServer, error) {
	var (
		serverID        int64
		instanceKey     string
		redisHost       string
		status          string
		startedAt       sql.NullTime
		lastHeartbeatAt sql.NullTime
		archivedAt      sql.NullTime
	)

	if err := row.Scan(
		&serverID,
		&instanceKey,
		&redisHost,
		&status,
		&startedAt,
		&lastHeartbeatAt,
		&archivedAt,
	); err != nil {
		return nil, err
	}

	server := &domain.GameServer{
		ServerID:    int(serverID),
		InstanceKey: instanceKey,
		RedisHost:   redisHost,
		Status:      status,
	}
	if startedAt.Valid {
		server.StartedAt = startedAt.Time.UTC()
	}
	if lastHeartbeatAt.Valid {
		server.LastHeartbeatAt = lastHeartbeatAt.Time.UTC()
	}
	if archivedAt.Valid {
		archivedCopy := archivedAt.Time.UTC()
		server.ArchivedAt = &archivedCopy
	}

	return server, nil
}
