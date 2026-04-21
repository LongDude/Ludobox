package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var roomSortableColumns = map[string]string{
	"room_id":                   "r.room_id",
	"config_id":                 "r.config_id",
	"server_id":                 "r.server_id",
	"server_name":               "gs.instance_key",
	"current_players":           "r.current_players",
	"status":                    "r.status",
	"archived_at":               "r.archived_at",
	"config_capacity":           "c.capacity",
	"config_registration_price": "c.registration_price",
	"config_is_boost":           "c.is_boost",
	"config_game_id":            "c.game_id",
	"config_game_name":          "g.name_game",
}

type roomFilterField struct {
	column string
	kind   string
}

var roomFilterColumns = map[string]roomFilterField{
	"room_id":                   {column: "r.room_id", kind: "int"},
	"config_id":                 {column: "r.config_id", kind: "int"},
	"server_id":                 {column: "r.server_id", kind: "int"},
	"server_name":               {column: "gs.instance_key", kind: "string"},
	"current_players":           {column: "r.current_players", kind: "int"},
	"status":                    {column: "r.status", kind: "string"},
	"config_capacity":           {column: "c.capacity", kind: "int"},
	"config_registration_price": {column: "c.registration_price", kind: "int"},
	"config_is_boost":           {column: "c.is_boost", kind: "bool"},
	"config_game_id":            {column: "c.game_id", kind: "int"},
	"config_game_name":          {column: "g.name_game", kind: "string"},
}

func (r *roomRepository) CreateRoomByConfigID(ctx context.Context, configID int, serverID int) (*domain.Room, error) {
	const query = `
		INSERT INTO rooms (config_id, server_id, status)
		VALUES ($1, $2, 'open'::room_status)
		RETURNING room_id
	`

	var roomID int64
	if err := r.db.QueryRow(ctx, query, configID, serverID).Scan(&roomID); err != nil {
		return nil, wrapRoomMutationError("create room by config id", err)
	}

	return r.GetRoomByID(ctx, int(roomID))
}

func (r *roomRepository) DeleteRoomByID(ctx context.Context, id int) error {
	const query = `
		UPDATE rooms
		SET archived_at = NOW()
		WHERE room_id = $1
		  AND archived_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("archive room by id: %w", err)
	}
	if result.RowsAffected() > 0 {
		return nil
	}

	found, archived, err := roomPresence(ctx, r.db, id)
	if err != nil {
		return fmt.Errorf("archive room by id: %w", err)
	}
	if !found {
		return repository.ErrorRoomNotFound
	}
	if archived {
		return repository.ErrorRoomArchived
	}

	return repository.ErrorRoomNotFound
}

func (r *roomRepository) GetNotArchivedRooms(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Room], error) {
	response := domain.ListResponse[domain.Room]{
		Items: make([]domain.Room, 0),
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	whereParts := []string{"r.archived_at IS NULL"}
	args := make([]any, 0, len(params.Filter)+2)
	for _, filter := range params.Filter {
		clause, clauseArgs, err := buildRoomFilterClause(filter, len(args)+1)
		if err != nil {
			return response, err
		}
		whereParts = append(whereParts, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(whereParts, " AND ")

	orderBy, err := buildRoomOrderBy(params.Sort)
	if err != nil {
		return response, err
	}

	countQuery := `
		SELECT COUNT(*)
		FROM rooms r
		JOIN config c ON c.config_id = r.config_id
		JOIN games g ON g.game_id = c.game_id
		LEFT JOIN game_servers gs ON gs.server_id = r.server_id
		WHERE ` + whereSQL
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&response.Total); err != nil {
		return response, fmt.Errorf("count rooms: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT
			r.room_id,
			r.config_id,
			r.server_id,
			r.current_players,
			r.status,
			r.archived_at,
			c.config_id,
			c.game_id,
			c.capacity,
			c.registration_price,
			c.is_boost,
			c.boost_price,
			c.boost_power,
			c.number_winners,
			c.winning_distribution,
			c.commission,
			c.time,
			c.min_users,
			c.archived_at,
			g.game_id,
			g.name_game,
			g.archived_at,
			gs.instance_key
		FROM rooms r
		JOIN config c ON c.config_id = r.config_id
		JOIN games g ON g.game_id = c.game_id
		LEFT JOIN game_servers gs ON gs.server_id = r.server_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderBy, len(args)+1, len(args)+2)

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, listQuery, append(args, pageSize, offset)...)
	if err != nil {
		return response, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		room, scanErr := scanRoom(rows)
		if scanErr != nil {
			return response, fmt.Errorf("scan room: %w", scanErr)
		}
		response.Items = append(response.Items, *room)
	}
	if err := rows.Err(); err != nil {
		return response, fmt.Errorf("iterate rooms: %w", err)
	}

	return response, nil
}

func (r *roomRepository) GetRoomByID(ctx context.Context, id int) (*domain.Room, error) {
	room, err := getRoomByID(ctx, r.db, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorRoomNotFound
		}
		return nil, fmt.Errorf("get room by id: %w", err)
	}

	return room, nil
}

func (r *roomRepository) UpdateRoomByID(ctx context.Context, id int, room *domain.Room) (*domain.Room, error) {
	const query = `
		UPDATE rooms
		SET server_id = $2,
			archived_at = $3
		WHERE room_id = $1
		RETURNING room_id
	`

	var updatedID int64
	if err := r.db.QueryRow(ctx, query, id, room.GameServerID, room.ArchivedAt).Scan(&updatedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorRoomNotFound
		}
		return nil, wrapRoomMutationError("update room by id", err)
	}

	return r.GetRoomByID(ctx, int(updatedID))
}

func buildRoomOrderBy(sort *domain.Sort) (string, error) {
	if sort == nil {
		return "r.room_id DESC", nil
	}

	column, ok := roomSortableColumns[strings.ToLower(strings.TrimSpace(sort.Field))]
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

func buildRoomFilterClause(filter domain.Filter, startIndex int) (string, []any, error) {
	fieldName := strings.ToLower(strings.TrimSpace(filter.Field))
	field, ok := roomFilterColumns[fieldName]
	if !ok {
		return "", nil, fmt.Errorf("%w: unsupported filter field %q", repository.ErrorInvalidListParams, filter.Field)
	}

	operatorName := strings.ToLower(strings.TrimSpace(filter.Operator))
	switch field.kind {
	case "int":
		if !isScalarOperator(operatorName) {
			return "", nil, fmt.Errorf("%w: unsupported filter operator %q for field %q", repository.ErrorInvalidListParams, filter.Operator, filter.Field)
		}
	case "bool":
		if !isBoolOperator(operatorName) {
			return "", nil, fmt.Errorf("%w: unsupported filter operator %q for field %q", repository.ErrorInvalidListParams, filter.Operator, filter.Field)
		}
	case "string":
		if !isStringOperator(operatorName) {
			return "", nil, fmt.Errorf("%w: unsupported filter operator %q for field %q", repository.ErrorInvalidListParams, filter.Operator, filter.Field)
		}
	default:
		return "", nil, fmt.Errorf("%w: unsupported filter field type", repository.ErrorInvalidListParams)
	}

	if operatorName == "in" || operatorName == "not_in" {
		values, err := convertRoomInValues(filter.Value, field.kind)
		if err != nil {
			return "", nil, err
		}
		placeholders := make([]string, 0, len(values))
		for i := range values {
			placeholders = append(placeholders, fmt.Sprintf("$%d", startIndex+i))
		}
		return fmt.Sprintf("%s %s (%s)", field.column, getOperator(operatorName), strings.Join(placeholders, ", ")), values, nil
	}

	value, err := convertRoomFilterValue(filter.Value, field.kind, operatorName)
	if err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("%s %s $%d", field.column, getOperator(operatorName), startIndex), []any{value}, nil
}

func convertRoomFilterValue(raw any, kind string, operator string) (any, error) {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" {
		return nil, fmt.Errorf("%w: filter value cannot be empty", repository.ErrorInvalidListParams)
	}

	switch kind {
	case "int":
		value, err := strconv.Atoi(text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid integer value %q", repository.ErrorInvalidListParams, text)
		}
		return value, nil
	case "bool":
		value, err := strconv.ParseBool(text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid boolean value %q", repository.ErrorInvalidListParams, text)
		}
		return value, nil
	case "string":
		if operator == "like" || operator == "not_like" {
			return "%" + text + "%", nil
		}
		return text, nil
	default:
		return nil, fmt.Errorf("%w: unsupported filter field type", repository.ErrorInvalidListParams)
	}
}

func convertRoomInValues(raw any, kind string) ([]any, error) {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" {
		return nil, fmt.Errorf("%w: filter value cannot be empty", repository.ErrorInvalidListParams)
	}

	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '|' || r == ';'
	})
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: filter value cannot be empty", repository.ErrorInvalidListParams)
	}

	values := make([]any, 0, len(parts))
	for _, part := range parts {
		value, err := convertRoomFilterValue(part, kind, "")
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

func roomPresence(ctx context.Context, db queryRower, id int) (bool, bool, error) {
	const query = `
		SELECT archived_at IS NOT NULL
		FROM rooms
		WHERE room_id = $1
	`

	var archived bool
	err := db.QueryRow(ctx, query, id).Scan(&archived)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, err
	}

	return true, archived, nil
}

func scanRoom(row rowScanner) (*domain.Room, error) {
	var (
		id                  int64
		configID            int64
		serverID            int64
		currentPlayers      int32
		status              string
		archivedAt          sql.NullTime
		joinedConfigID      int64
		gameID              int64
		capacity            int32
		registrationPrice   int64
		isBoost             bool
		boostPrice          int64
		boostPower          int32
		numberWinners       int32
		winningDistribution []int32
		commission          int32
		roundTime           int32
		minUsers            int32
		configArchivedAt    sql.NullTime
		joinedGameID        int64
		gameName            string
		gameArchivedAt      sql.NullTime
		serverName          sql.NullString
	)

	if err := row.Scan(
		&id,
		&configID,
		&serverID,
		&currentPlayers,
		&status,
		&archivedAt,
		&joinedConfigID,
		&gameID,
		&capacity,
		&registrationPrice,
		&isBoost,
		&boostPrice,
		&boostPower,
		&numberWinners,
		&winningDistribution,
		&commission,
		&roundTime,
		&minUsers,
		&configArchivedAt,
		&joinedGameID,
		&gameName,
		&gameArchivedAt,
		&serverName,
	); err != nil {
		return nil, err
	}

	room := &domain.Room{
		ID:             int(id),
		ConfigID:       int(configID),
		GameServerID:   int(serverID),
		CurrentPlayers: int(currentPlayers),
		Status:         domain.RoomStatus(status),
		Config: &domain.Config{
			ID:                  int(joinedConfigID),
			GameID:              int(gameID),
			Capacity:            int(capacity),
			RegistrationPrice:   int(registrationPrice),
			IsBoost:             isBoost,
			BoostPrice:          int(boostPrice),
			BoostPower:          int(boostPower),
			NumberWinners:       int(numberWinners),
			WinningDistribution: toIntSlice(winningDistribution),
			Commission:          int(commission),
			Time:                int(roundTime),
			MinUsers:            int(minUsers),
			Game: &domain.Game{
				ID:   int(joinedGameID),
				Name: gameName,
			},
		},
	}
	if archivedAt.Valid {
		archivedCopy := archivedAt.Time.UTC()
		room.ArchivedAt = &archivedCopy
	}
	if configArchivedAt.Valid && room.Config != nil {
		archivedCopy := configArchivedAt.Time.UTC()
		room.Config.ArchivedAt = &archivedCopy
	}
	if gameArchivedAt.Valid && room.Config != nil && room.Config.Game != nil {
		archivedCopy := gameArchivedAt.Time.UTC()
		room.Config.Game.ArchivedAt = &archivedCopy
	}
	if serverName.Valid {
		room.ServerName = serverName.String
	}

	return room, nil
}

func getRoomByID(ctx context.Context, db queryRower, id int) (*domain.Room, error) {
	const query = `
		SELECT
			r.room_id,
			r.config_id,
			r.server_id,
			r.current_players,
			r.status,
			r.archived_at,
			c.config_id,
			c.game_id,
			c.capacity,
			c.registration_price,
			c.is_boost,
			c.boost_price,
			c.boost_power,
			c.number_winners,
			c.winning_distribution,
			c.commission,
			c.time,
			c.min_users,
			c.archived_at,
			g.game_id,
			g.name_game,
			g.archived_at,
			gs.instance_key
		FROM rooms r
		JOIN config c ON c.config_id = r.config_id
		JOIN games g ON g.game_id = c.game_id
		LEFT JOIN game_servers gs ON gs.server_id = r.server_id
		WHERE r.room_id = $1
	`

	return scanRoom(db.QueryRow(ctx, query, id))
}

func wrapRoomMutationError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503", "23514", "22P02":
			return fmt.Errorf("%w: %s", repository.ErrorInvalidRoom, pgErr.Message)
		}
	}

	return fmt.Errorf("%s: %w", action, err)
}
