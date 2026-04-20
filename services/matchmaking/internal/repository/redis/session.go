package redis

import (
	"context"
	"time"
	"user_service/internal/domain"
	"user_service/internal/repository"
)

func (s *sessionRepository) GetRoomRecommendations(ctx context.Context, key string) ([]domain.RoomRecommendation, error) {
	exists, err := s.redis.Exists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrorCacheMiss
	}

	var recommendations []domain.RoomRecommendation
	if err := s.redis.Get(ctx, key, &recommendations); err != nil {
		return nil, err
	}

	return recommendations, nil
}

func (s *sessionRepository) SetRoomRecommendations(ctx context.Context, key string, recommendations []domain.RoomRecommendation, ttl time.Duration) error {
	return s.redis.Set(ctx, key, recommendations, ttl)
}

func (s *sessionRepository) DeleteKey(ctx context.Context, key string) error {
	return s.redis.Delete(ctx, key)
}

func (s *sessionRepository) DeleteByPrefix(ctx context.Context, prefix string) error {
	return s.redis.DeleteByPrefix(ctx, prefix)
}
