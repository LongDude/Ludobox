package service

import (
	"context"
	"fmt"
	"strings"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type GameServerService interface {
	GetServers(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.GameServer], error)
}

type gameServerService struct {
	gameServerRepository repository.GameServerRepository
	logger               *logrus.Logger
}

func NewGameServerService(gameServerRepository repository.GameServerRepository, logger *logrus.Logger) GameServerService {
	return &gameServerService{
		gameServerRepository: gameServerRepository,
		logger:               logger,
	}
}

func (s *gameServerService) GetServers(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.GameServer], error) {
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
			Field:     "server_id",
			Direction: "desc",
		}
	} else {
		params.Sort.Field = strings.TrimSpace(params.Sort.Field)
		params.Sort.Direction = strings.ToLower(strings.TrimSpace(params.Sort.Direction))
		if params.Sort.Field == "" {
			return domain.ListResponse[domain.GameServer]{}, fmt.Errorf("%w: sort field cannot be empty", repository.ErrorInvalidListParams)
		}
		if params.Sort.Direction == "" {
			params.Sort.Direction = "asc"
		}
	}

	return s.gameServerRepository.GetServers(ctx, params)
}
