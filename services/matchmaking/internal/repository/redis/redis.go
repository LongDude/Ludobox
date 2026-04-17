package redis

import (
	"user_service/internal/repository"
	"user_service/pkg/storage"
)

type sessionRepository struct {
	redis *storage.RedisClient
}

func NewSessionRepository(redis *storage.RedisClient) repository.SessionRepository {
	return &sessionRepository{redis: redis}
}
