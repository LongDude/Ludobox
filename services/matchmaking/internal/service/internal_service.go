package service

import (
	"context"
	"time"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type InternalService interface {
	ResolveRoomOwner(ctx context.Context, roomID int64, staleAfter time.Duration) (*domain.GameServer, error)
	SelectAvailableGameServer(ctx context.Context, staleAfter time.Duration) (*domain.GameServer, error)
	RecommendRooms(ctx context.Context, preferences domain.MatchmakingPreferences) (domain.ListResponse[domain.RoomRecommendation], bool, error)
	QuickMatch(ctx context.Context, preferences domain.MatchmakingPreferences) (*domain.QuickMatchResult, error)
}

type internalService struct {
	internalRepository repository.InternalRepository
	sessionRepository  repository.SessionRepository
	recommendationTTL  time.Duration
	logger             *logrus.Logger
}

func NewInternalService(
	internalRepository repository.InternalRepository,
	sessionRepository repository.SessionRepository,
	recommendationTTL time.Duration,
	logger *logrus.Logger,
) InternalService {
	return &internalService{
		internalRepository: internalRepository,
		sessionRepository:  sessionRepository,
		recommendationTTL:  recommendationTTL,
		logger:             logger,
	}
}

func (s *internalService) ResolveRoomOwner(ctx context.Context, roomID int64, staleAfter time.Duration) (*domain.GameServer, error) {
	gameServer, err := s.internalRepository.ResolveRoomOwner(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if gameServer.ArchivedAt != nil ||
		gameServer.Status != "up" ||
		time.Since(gameServer.LastHeartbeatAt) > staleAfter {
		return nil, domain.ErrorGameServerUnavailable
	}

	return gameServer, nil
}

func (s *internalService) SelectAvailableGameServer(ctx context.Context, staleAfter time.Duration) (*domain.GameServer, error) {
	return s.internalRepository.SelectAvailableGameServer(ctx, staleAfter)
}
