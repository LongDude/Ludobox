package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/repository"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

func gameIDFromPath(ctx *gin.Context) (int, error) {
	raw := strings.TrimSpace(ctx.Param("game_id"))
	if raw == "" {
		return 0, fmt.Errorf("game_id is required")
	}

	gameID, err := strconv.Atoi(raw)
	if err != nil || gameID <= 0 {
		return 0, fmt.Errorf("game_id must be a positive integer")
	}

	return gameID, nil
}

func gameToResponse(game *domain.Game) *presenters.GameResponse {
	if game == nil {
		return nil
	}

	return &presenters.GameResponse{
		GameID:     game.ID,
		NameGame:   game.Name,
		ArchivedAt: game.ArchivedAt,
	}
}

func gameRequestToDomain(req presenters.GameUpsertRequest) *domain.Game {
	return &domain.Game{
		Name: req.NameGame,
	}
}

// GetGameByID godoc
// @Summary Get game by id
// @Description Returns a single game by id.
// @Tags Game
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param game_id path int true "Game id"
// @Success 200 {object} presenters.GameResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/game/{game_id} [get]
func GetGameByID(ctx *gin.Context, a *app.App) {
	gameID, err := gameIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	game, err := a.GameService.GetGameByID(ctx, gameID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorGameNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorInvalidListParams):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("get game failed: %w", err)))
		}
		return
	}

	ctx.JSON(http.StatusOK, gameToResponse(game))
}

// GetGames godoc
// @Summary List active games
// @Description Returns non-archived games with pagination and sorting.
// @Tags Game
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param sort_field query string false "Sort field"
// @Param sort_direction query string false "Sort direction (asc/desc)"
// @Success 200 {object} presenters.GamesResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/games [get]
func GetGames(ctx *gin.Context, a *app.App) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	sortField := strings.TrimSpace(ctx.Query("sort_field"))
	sortDirection := strings.TrimSpace(ctx.Query("sort_direction"))

	params := domain.ListParams{
		Pagination: domain.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
	}
	if sortField != "" {
		params.Sort = &domain.Sort{
			Field:     sortField,
			Direction: sortDirection,
		}
	}

	games, err := a.GameService.GetGames(ctx, params)
	if err != nil {
		if errors.Is(err, repository.ErrorInvalidListParams) {
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("list games failed: %w", err)))
		return
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items := make([]presenters.GameResponse, 0, len(games.Items))
	for i := range games.Items {
		response := gameToResponse(&games.Items[i])
		if response != nil {
			items = append(items, *response)
		}
	}

	ctx.JSON(http.StatusOK, presenters.GamesResponse{
		Items:    items,
		Total:    games.Total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CreateGame godoc
// @Summary Create game
// @Description Creates a new game.
// @Tags Game
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param request body presenters.GameUpsertRequest true "Game payload"
// @Success 201 {object} presenters.GameResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/game [post]
func CreateGame(ctx *gin.Context, a *app.App) {
	var req presenters.GameUpsertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}

	game, err := a.GameService.CreateGame(ctx, gameRequestToDomain(req))
	if err != nil {
		if errors.Is(err, repository.ErrorInvalidGame) {
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("create game failed: %w", err)))
		return
	}

	ctx.JSON(http.StatusCreated, gameToResponse(game))
}

// UpdateGameByID godoc
// @Summary Update game by id
// @Description Updates an active game by id.
// @Tags Game
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param game_id path int true "Game id"
// @Param request body presenters.GameUpsertRequest true "Game payload"
// @Success 200 {object} presenters.GameResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/game/{game_id} [put]
func UpdateGameByID(ctx *gin.Context, a *app.App) {
	gameID, err := gameIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	var req presenters.GameUpsertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}

	game, err := a.GameService.UpdateGameByID(ctx, gameID, gameRequestToDomain(req))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorInvalidGame), errors.Is(err, repository.ErrorInvalidListParams):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		case errors.Is(err, repository.ErrorGameNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorGameArchived):
			ctx.JSON(http.StatusConflict, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("update game failed: %w", err)))
		}
		return
	}

	ctx.JSON(http.StatusOK, gameToResponse(game))
}

// DeleteGameByID godoc
// @Summary Archive game
// @Description Archives an active game by id.
// @Tags Game
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param game_id path int true "Game id"
// @Success 204
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/game/{game_id} [delete]
func DeleteGameByID(ctx *gin.Context, a *app.App) {
	gameID, err := gameIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	err = a.GameService.DeleteGameByID(ctx, gameID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorInvalidListParams):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		case errors.Is(err, repository.ErrorGameNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorGameArchived):
			ctx.JSON(http.StatusConflict, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("delete game failed: %w", err)))
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}
