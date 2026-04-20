package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var gameSortableColumns = map[string]string{
	"game_id":     "game_id",
	"name_game":   "name_game",
	"archived_at": "archived_at",
}

func (g *gameRepository) GetGames(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Game], error) {
	response := domain.ListResponse[domain.Game]{
		Items: make([]domain.Game, 0),
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	whereSQL := "archived_at IS NULL"
	orderBy, err := buildGameOrderBy(params.Sort)
	if err != nil {
		return response, err
	}

	const countQuery = `
		SELECT COUNT(*)
		FROM games
		WHERE archived_at IS NULL
	`
	if err := g.db.QueryRow(ctx, countQuery).Scan(&response.Total); err != nil {
		return response, fmt.Errorf("count games: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT
			game_id,
			name_game,
			archived_at
		FROM games
		WHERE %s
		ORDER BY %s
		LIMIT $1 OFFSET $2
	`, whereSQL, orderBy)

	offset := (page - 1) * pageSize
	rows, err := g.db.Query(ctx, listQuery, pageSize, offset)
	if err != nil {
		return response, fmt.Errorf("list games: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		game, scanErr := scanGame(rows)
		if scanErr != nil {
			return response, fmt.Errorf("scan game: %w", scanErr)
		}
		response.Items = append(response.Items, *game)
	}
	if err := rows.Err(); err != nil {
		return response, fmt.Errorf("iterate games: %w", err)
	}

	return response, nil
}

func (g *gameRepository) GetGameByID(ctx context.Context, id int) (*domain.Game, error) {
	const query = `
		SELECT
			game_id,
			name_game,
			archived_at
		FROM games
		WHERE game_id = $1
	`

	game, err := scanGame(g.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrorGameNotFound
		}
		return nil, fmt.Errorf("get game by id: %w", err)
	}

	return game, nil
}

func (g *gameRepository) CreateGame(ctx context.Context, game *domain.Game) (*domain.Game, error) {
	const query = `
		INSERT INTO games (name_game)
		VALUES ($1)
		RETURNING game_id, name_game, archived_at
	`

	created, err := scanGame(g.db.QueryRow(ctx, query, strings.TrimSpace(game.Name)))
	if err != nil {
		return nil, wrapGameMutationError("create game", err)
	}

	return created, nil
}

func (g *gameRepository) UpdateGameByID(ctx context.Context, id int, game *domain.Game) (*domain.Game, error) {
	const query = `
		UPDATE games
		SET name_game = $2
		WHERE game_id = $1
		  AND archived_at IS NULL
		RETURNING game_id, name_game, archived_at
	`

	updated, err := scanGame(g.db.QueryRow(ctx, query, id, strings.TrimSpace(game.Name)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			found, archived, presenceErr := gamePresence(ctx, g.db, id)
			if presenceErr != nil {
				return nil, fmt.Errorf("update game by id: %w", presenceErr)
			}
			if !found {
				return nil, repository.ErrorGameNotFound
			}
			if archived {
				return nil, repository.ErrorGameArchived
			}
		}
		return nil, wrapGameMutationError("update game by id", err)
	}

	return updated, nil
}

func (g *gameRepository) DeleteGameByID(ctx context.Context, id int) error {
	const query = `
		UPDATE games
		SET archived_at = NOW()
		WHERE game_id = $1
		  AND archived_at IS NULL
	`

	result, err := g.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("archive game by id: %w", err)
	}
	if result.RowsAffected() > 0 {
		return nil
	}

	found, archived, err := gamePresence(ctx, g.db, id)
	if err != nil {
		return fmt.Errorf("archive game by id: %w", err)
	}
	if !found {
		return repository.ErrorGameNotFound
	}
	if archived {
		return repository.ErrorGameArchived
	}

	return repository.ErrorGameNotFound
}

func buildGameOrderBy(sort *domain.Sort) (string, error) {
	if sort == nil {
		return "game_id DESC", nil
	}

	column, ok := gameSortableColumns[strings.ToLower(strings.TrimSpace(sort.Field))]
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

func gamePresence(ctx context.Context, db queryRower, id int) (bool, bool, error) {
	const query = `
		SELECT archived_at IS NOT NULL
		FROM games
		WHERE game_id = $1
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

func scanGame(row rowScanner) (*domain.Game, error) {
	var (
		id         int64
		name       string
		archivedAt sql.NullTime
	)

	if err := row.Scan(&id, &name, &archivedAt); err != nil {
		return nil, err
	}

	game := &domain.Game{
		ID:   int(id),
		Name: name,
	}
	if archivedAt.Valid {
		archivedCopy := archivedAt.Time.UTC()
		game.ArchivedAt = &archivedCopy
	}

	return game, nil
}

func wrapGameMutationError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23514", "22P02":
			return fmt.Errorf("%w: %s", repository.ErrorInvalidGame, pgErr.Message)
		}
	}

	return fmt.Errorf("%s: %w", action, err)
}
