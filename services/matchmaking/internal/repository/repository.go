package repository

import (
	"context"
	"errors"
	"time"
	"user_service/internal/domain"
)

var (
	ErrorCacheMiss = errors.New("cache miss")
)

type InternalRepository interface {
	ResolveRoomOwner(ctx context.Context, roomID int64) (*domain.GameServer, error)
	SelectAvailableGameServer(ctx context.Context, staleAfter time.Duration) (*domain.GameServer, error)
	RecommendRooms(ctx context.Context, preferences domain.MatchmakingPreferences) ([]domain.RoomRecommendation, error)
	GetUserActiveRoom(ctx context.Context, userID int64) (*domain.RoomMembership, error)
	JoinRoom(ctx context.Context, userID int64, roomID int64) (*domain.RoomMembership, error)
}

type SessionRepository interface {
	GetRoomRecommendations(ctx context.Context, key string) ([]domain.RoomRecommendation, error)
	SetRoomRecommendations(ctx context.Context, key string, recommendations []domain.RoomRecommendation, ttl time.Duration) error
	DeleteKey(ctx context.Context, key string) error
}
