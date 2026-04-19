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
	"room_id":     "room_id",
	"config_id":   "config_id",
	"server_id":   "server_id",
	"status":      "status",
	"archived_at": "archived_at",
}

type roomFilterField struct {
	column string
	kind   string
}

var roomFilterColumns = map[string]roomFilterField{
	"room_id":   {column: "room_id", kind: "int"},
	"config_id": {column: "config_id", kind: "int"},
	"server_id": {column: "server_id", kind: "int"},
	"status":    {column: "status", kind: "string"},
}

func (r *roomRepository) CreateRoomByConfigID(ctx context.Context, configID int, serverID int) (*domain.Room, error) {
	const query = `
		INSERT INTO rooms (config_id, server_id, status)
		VALUES ($1, $2, 'open'::room_status)
		RETURNING room_id, config_id, server_id, status, archived_at
	`

	room, err := scanRoom(r.db.QueryRow(ctx, query, configID, serverID))
	if err != nil {
		return nil, wrapRoomMutationError("create room by config id", err)
	}

	return room, nil
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

	whereParts := []string{"archived_at IS NULL"}
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

	countQuery := "SELECT COUNT(*) FROM rooms WHERE " + whereSQL
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&response.Total); err != nil {
		return response, fmt.Errorf("count rooms: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT
			room_id,
			config_id,
			server_id,
			status,
			archived_at
		FROM rooms
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
	const query = `
		SELECT
			room_id,
			config_id,
			server_id,
			status,
			archived_at
		FROM rooms
		WHERE room_id = $1
	`

	room, err := scanRoom(r.db.QueryRow(ctx, query, id))
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
		RETURNING room_id, config_id, server_id, status, archived_at
	`

	updated, err := scanRoom(r.db.QueryRow(ctx, query, id, room.GameServerID, room.ArchivedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorRoomNotFound
		}
		return nil, wrapRoomMutationError("update room by id", err)
	}

	return updated, nil
}

func buildRoomOrderBy(sort *domain.Sort) (string, error) {
	if sort == nil {
		return "room_id DESC", nil
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
		id         int64
		configID   int64
		serverID   int64
		status     string
		archivedAt sql.NullTime
	)

	if err := row.Scan(&id, &configID, &serverID, &status, &archivedAt); err != nil {
		return nil, err
	}

	room := &domain.Room{
		ID:           int(id),
		ConfigID:     int(configID),
		GameServerID: int(serverID),
		Status:       domain.RoomStatus(status),
	}
	if archivedAt.Valid {
		archivedCopy := archivedAt.Time.UTC()
		room.ArchivedAt = &archivedCopy
	}

	return room, nil
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
