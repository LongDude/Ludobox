package redis

import (
	"game_server/internal/repository"
	"game_server/pkg/storage"
)

type sessionRepository struct {
	redis *storage.RedisClient
}

func NewSessionRepository(redis *storage.RedisClient) repository.SessionRepository {
	return &sessionRepository{redis: redis}
}
