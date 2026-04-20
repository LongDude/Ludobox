package service

import (
	"context"
	"fmt"
	"strings"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type GameService interface {
	GetGames(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Game], error)
	GetGameByID(ctx context.Context, id int) (*domain.Game, error)
	CreateGame(ctx context.Context, game *domain.Game) (*domain.Game, error)
	UpdateGameByID(ctx context.Context, id int, game *domain.Game) (*domain.Game, error)
	DeleteGameByID(ctx context.Context, id int) error
}

type gameService struct {
	gameRepository repository.GameRepository
	logger         *logrus.Logger
}

func NewGameService(gameRepository repository.GameRepository, logger *logrus.Logger) GameService {
	return &gameService{
		gameRepository: gameRepository,
		logger:         logger,
	}
}

func (s *gameService) GetGames(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Game], error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	if params.Sort == nil {
		params.Sort = &domain.Sort{
			Field:     "game_id",
			Direction: "desc",
		}
	} else {
		params.Sort.Field = strings.TrimSpace(params.Sort.Field)
		params.Sort.Direction = strings.ToLower(strings.TrimSpace(params.Sort.Direction))
		if params.Sort.Field == "" {
			return domain.ListResponse[domain.Game]{}, fmt.Errorf("%w: sort field cannot be empty", repository.ErrorInvalidListParams)
		}
		if params.Sort.Direction == "" {
			params.Sort.Direction = "asc"
		}
	}

	return s.gameRepository.GetGames(ctx, params)
}

func (s *gameService) GetGameByID(ctx context.Context, id int) (*domain.Game, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: game_id must be positive", repository.ErrorInvalidListParams)
	}

	return s.gameRepository.GetGameByID(ctx, id)
}

func (s *gameService) CreateGame(ctx context.Context, game *domain.Game) (*domain.Game, error) {
	if err := validateGame(game); err != nil {
		return nil, err
	}

	return s.gameRepository.CreateGame(ctx, game)
}

func (s *gameService) UpdateGameByID(ctx context.Context, id int, game *domain.Game) (*domain.Game, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: game_id must be positive", repository.ErrorInvalidListParams)
	}
	if err := validateGame(game); err != nil {
		return nil, err
	}

	return s.gameRepository.UpdateGameByID(ctx, id, game)
}

func (s *gameService) DeleteGameByID(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("%w: game_id must be positive", repository.ErrorInvalidListParams)
	}

	return s.gameRepository.DeleteGameByID(ctx, id)
}

func validateGame(game *domain.Game) error {
	if game == nil {
		return fmt.Errorf("%w: request body is required", repository.ErrorInvalidGame)
	}
	if strings.TrimSpace(game.Name) == "" {
		return fmt.Errorf("%w: name_game cannot be empty", repository.ErrorInvalidGame)
	}

	return nil
}
