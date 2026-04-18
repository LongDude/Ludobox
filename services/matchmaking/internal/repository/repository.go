package repository

import (
	"context"
	"time"
	"user_service/internal/domain"
)

type InternalRepository interface {
	ResolveRoomOwner(ctx context.Context, roomID int64) (*domain.GameServer, error)
	SelectAvailableGameServer(ctx context.Context, staleAfter time.Duration) (*domain.GameServer, error)
}
type SessionRepository interface {
}
