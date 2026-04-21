package service

import (
	"context"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type GameHistoryService interface {
	GetUserGameHistory(ctx context.Context, userID int, params domain.GameHistoryParams) (domain.ListResponse[domain.GameHistoryItem], error)
}

type gameHistoryService struct {
	repository repository.GameHistoryRepository
	logger     *logrus.Logger
}

func NewGameHistoryService(repository repository.GameHistoryRepository, logger *logrus.Logger) GameHistoryService {
	return &gameHistoryService{
		repository: repository,
		logger:     logger,
	}
}

func (s *gameHistoryService) GetUserGameHistory(ctx context.Context, userID int, params domain.GameHistoryParams) (domain.ListResponse[domain.GameHistoryItem], error) {
	return s.repository.GetUserGameHistory(ctx, userID, params)
}
