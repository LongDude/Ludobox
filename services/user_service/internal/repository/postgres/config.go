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

type rowScanner interface {
	Scan(dest ...any) error
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var configSortableColumns = map[string]string{
	"config_id":          "config_id",
	"game_id":            "game_id",
	"capacity":           "capacity",
	"registration_price": "registration_price",
	"is_boost":           "is_boost",
	"boost_price":        "boost_price",
	"boost_power":        "boost_power",
	"number_winners":     "number_winners",
	"commission":         "commission",
	"time":               "time",
	"min_users":          "min_users",
	"archived_at":        "archived_at",
}

type configFilterField struct {
	column string
	kind   string
}

var configFilterColumns = map[string]configFilterField{
	"config_id":            {column: "config_id", kind: "int"},
	"game_id":              {column: "game_id", kind: "int"},
	"capacity":             {column: "capacity", kind: "int"},
	"registration_price":   {column: "registration_price", kind: "int"},
	"is_boost":             {column: "is_boost", kind: "bool"},
	"boost_price":          {column: "boost_price", kind: "int"},
	"boost_power":          {column: "boost_power", kind: "int"},
	"number_winners":       {column: "number_winners", kind: "int"},
	"winning_distribution": {column: "winning_distribution", kind: "int_array"},
	"commission":           {column: "commission", kind: "int"},
	"time":                 {column: "time", kind: "int"},
	"min_users":            {column: "min_users", kind: "int"},
}

var supportedScalarOperators = map[string]struct{}{
	"eq":     {},
	"neq":    {},
	"gt":     {},
	"lt":     {},
	"gte":    {},
	"lte":    {},
	"in":     {},
	"not_in": {},
}

var supportedBoolOperators = map[string]struct{}{
	"eq":     {},
	"neq":    {},
	"in":     {},
	"not_in": {},
}

var supportedArrayOperators = map[string]struct{}{
	"contains":  {},
	"contained": {},
	"overlap":   {},
}

func (c *configRepository) CreateNewConfig(ctx context.Context, config *domain.Config) (*domain.Config, error) {
	const query = `
		INSERT INTO config (
			game_id,
			capacity,
			registration_price,
			is_boost,
			boost_price,
			boost_power,
			number_winners,
			winning_distribution,
			commission,
			time,
			min_users
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING
			config_id,
			game_id,
			capacity,
			registration_price,
			is_boost,
			boost_price,
			boost_power,
			number_winners,
			winning_distribution,
			commission,
			time,
			min_users,
			archived_at
	`

	created, err := scanConfig(c.db.QueryRow(
		ctx,
		query,
		int64(config.GameID),
		config.Capacity,
		int64(config.RegistrationPrice),
		config.IsBoost,
		int64(config.BoostPrice),
		config.BoostPower,
		config.NumberWinners,
		toInt32Slice(config.WinningDistribution),
		config.Commission,
		config.Time,
		config.MinUsers,
	))
	if err != nil {
		return nil, wrapConfigMutationError("create config", err)
	}

	return created, nil
}

func (c *configRepository) DeleteConfigByID(ctx context.Context, id int) error {
	const archiveQuery = `
		UPDATE config
		SET archived_at = NOW()
		WHERE config_id = $1
		  AND archived_at IS NULL
	`

	result, err := c.db.Exec(ctx, archiveQuery, id)
	if err != nil {
		return fmt.Errorf("archive config by id: %w", err)
	}
	if result.RowsAffected() > 0 {
		return nil
	}

	found, archived, err := configPresence(ctx, c.db, id)
	if err != nil {
		return fmt.Errorf("archive config by id: %w", err)
	}
	if !found {
		return repository.ErrorConfigNotFound
	}
	if archived {
		return repository.ErrorConfigArchived
	}

	return repository.ErrorConfigNotFound
}

func (c *configRepository) GetConfigByID(ctx context.Context, id int) (*domain.Config, error) {
	const query = `
		SELECT
			config_id,
			game_id,
			capacity,
			registration_price,
			is_boost,
			boost_price,
			boost_power,
			number_winners,
			winning_distribution,
			commission,
			time,
			min_users,
			archived_at
		FROM config
		WHERE config_id = $1
	`

	config, err := scanConfig(c.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorConfigNotFound
		}
		return nil, fmt.Errorf("get config by id: %w", err)
	}

	return config, nil
}

func (c *configRepository) GetConfigs(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Config], error) {
	response := domain.ListResponse[domain.Config]{
		Items: make([]domain.Config, 0),
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
		clause, clauseArgs, err := buildConfigFilterClause(filter, len(args)+1)
		if err != nil {
			return response, err
		}
		whereParts = append(whereParts, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(whereParts, " AND ")

	orderBy, err := buildConfigOrderBy(params.Sort)
	if err != nil {
		return response, err
	}

	countQuery := "SELECT COUNT(*) FROM config WHERE " + whereSQL
	if err := c.db.QueryRow(ctx, countQuery, args...).Scan(&response.Total); err != nil {
		return response, fmt.Errorf("count configs: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT
			config_id,
			game_id,
			capacity,
			registration_price,
			is_boost,
			boost_price,
			boost_power,
			number_winners,
			winning_distribution,
			commission,
			time,
			min_users,
			archived_at
		FROM config
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderBy, len(args)+1, len(args)+2)

	offset := (page - 1) * pageSize
	rows, err := c.db.Query(ctx, listQuery, append(args, pageSize, offset)...)
	if err != nil {
		return response, fmt.Errorf("list configs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		config, scanErr := scanConfig(rows)
		if scanErr != nil {
			return response, fmt.Errorf("scan config: %w", scanErr)
		}
		response.Items = append(response.Items, *config)
	}
	if err := rows.Err(); err != nil {
		return response, fmt.Errorf("iterate configs: %w", err)
	}

	return response, nil
}

func (c *configRepository) UpdateConfigByID(ctx context.Context, id int, config *domain.Config) (*domain.Config, error) {
	tx, err := c.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin update config transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const archiveQuery = `
		UPDATE config
		SET archived_at = NOW()
		WHERE config_id = $1
		  AND archived_at IS NULL
	`

	result, err := tx.Exec(ctx, archiveQuery, id)
	if err != nil {
		return nil, fmt.Errorf("archive config before replace: %w", err)
	}
	if result.RowsAffected() == 0 {
		found, archived, presenceErr := configPresence(ctx, tx, id)
		if presenceErr != nil {
			return nil, fmt.Errorf("check config presence before replace: %w", presenceErr)
		}
		if !found {
			return nil, repository.ErrorConfigNotFound
		}
		if archived {
			return nil, repository.ErrorConfigArchived
		}

		return nil, repository.ErrorConfigNotFound
	}

	const insertQuery = `
		INSERT INTO config (
			game_id,
			capacity,
			registration_price,
			is_boost,
			boost_price,
			boost_power,
			number_winners,
			winning_distribution,
			commission,
			time,
			min_users
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING
			config_id,
			game_id,
			capacity,
			registration_price,
			is_boost,
			boost_price,
			boost_power,
			number_winners,
			winning_distribution,
			commission,
			time,
			min_users,
			archived_at
	`

	updated, err := scanConfig(tx.QueryRow(
		ctx,
		insertQuery,
		int64(config.GameID),
		config.Capacity,
		int64(config.RegistrationPrice),
		config.IsBoost,
		int64(config.BoostPrice),
		config.BoostPower,
		config.NumberWinners,
		toInt32Slice(config.WinningDistribution),
		config.Commission,
		config.Time,
		config.MinUsers,
	))
	if err != nil {
		return nil, wrapConfigMutationError("replace config", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update config transaction: %w", err)
	}

	return updated, nil
}

func buildConfigOrderBy(sort *domain.Sort) (string, error) {
	if sort == nil {
		return "config_id DESC", nil
	}

	column, ok := configSortableColumns[strings.ToLower(strings.TrimSpace(sort.Field))]
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

func buildConfigFilterClause(filter domain.Filter, startIndex int) (string, []any, error) {
	fieldName := strings.ToLower(strings.TrimSpace(filter.Field))
	field, ok := configFilterColumns[fieldName]
	if !ok {
		return "", nil, fmt.Errorf("%w: unsupported filter field %q", repository.ErrorInvalidListParams, filter.Field)
	}

	operatorName := strings.ToLower(strings.TrimSpace(filter.Operator))
	switch field.kind {
	case "int":
		if _, ok := supportedScalarOperators[operatorName]; !ok {
			return "", nil, fmt.Errorf("%w: unsupported filter operator %q for field %q", repository.ErrorInvalidListParams, filter.Operator, filter.Field)
		}
	case "bool":
		if _, ok := supportedBoolOperators[operatorName]; !ok {
			return "", nil, fmt.Errorf("%w: unsupported filter operator %q for field %q", repository.ErrorInvalidListParams, filter.Operator, filter.Field)
		}
	case "int_array":
		if _, ok := supportedArrayOperators[operatorName]; !ok {
			return "", nil, fmt.Errorf("%w: unsupported filter operator %q for field %q", repository.ErrorInvalidListParams, filter.Operator, filter.Field)
		}
	default:
		return "", nil, fmt.Errorf("%w: unsupported filter field type", repository.ErrorInvalidListParams)
	}

	if operatorName == "in" || operatorName == "not_in" {
		values, err := convertConfigInValues(filter.Value, field.kind)
		if err != nil {
			return "", nil, err
		}
		placeholders := make([]string, 0, len(values))
		for i := range values {
			placeholders = append(placeholders, fmt.Sprintf("$%d", startIndex+i))
		}
		return fmt.Sprintf("%s %s (%s)", field.column, getOperator(operatorName), strings.Join(placeholders, ", ")), values, nil
	}
	if field.kind == "int_array" {
		value, err := convertConfigArrayFilterValue(filter.Value)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("%s %s $%d", field.column, getOperatorArray(operatorName), startIndex), []any{value}, nil
	}

	operator := getOperator(operatorName)
	value, err := convertConfigFilterValue(filter.Value, field.kind)
	if err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("%s %s $%d", field.column, operator, startIndex), []any{value}, nil
}

func convertConfigFilterValue(raw any, kind string) (any, error) {
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
	default:
		return nil, fmt.Errorf("%w: unsupported filter field type", repository.ErrorInvalidListParams)
	}
}

func convertConfigInValues(raw any, kind string) ([]any, error) {
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
		value, err := convertConfigFilterValue(part, kind)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

func convertConfigArrayFilterValue(raw any) ([]int32, error) {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" {
		return nil, fmt.Errorf("%w: filter value cannot be empty", repository.ErrorInvalidListParams)
	}

	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '|' || r == ';' || r == ','
	})
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: filter value cannot be empty", repository.ErrorInvalidListParams)
	}

	result := make([]int32, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid integer value %q", repository.ErrorInvalidListParams, part)
		}
		result = append(result, int32(value))
	}

	return result, nil
}

func configPresence(ctx context.Context, db queryRower, id int) (bool, bool, error) {
	const query = `
		SELECT archived_at IS NOT NULL
		FROM config
		WHERE config_id = $1
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

func scanConfig(row rowScanner) (*domain.Config, error) {
	var (
		id                  int64
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
		archivedAt          sql.NullTime
	)

	if err := row.Scan(
		&id,
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
		&archivedAt,
	); err != nil {
		return nil, err
	}

	config := &domain.Config{
		ID:                  int(id),
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
	}
	if archivedAt.Valid {
		archivedCopy := archivedAt.Time.UTC()
		config.ArchivedAt = &archivedCopy
	}

	return config, nil
}

func wrapConfigMutationError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503", "23514", "22P02":
			return fmt.Errorf("%w: %s", repository.ErrorInvalidConfig, pgErr.Message)
		}
	}

	return fmt.Errorf("%s: %w", action, err)
}

func toInt32Slice(values []int) []int32 {
	if len(values) == 0 {
		return []int32{}
	}

	result := make([]int32, 0, len(values))
	for _, value := range values {
		result = append(result, int32(value))
	}

	return result
}

func toIntSlice(values []int32) []int {
	if len(values) == 0 {
		return []int{}
	}

	result := make([]int, 0, len(values))
	for _, value := range values {
		result = append(result, int(value))
	}

	return result
}
