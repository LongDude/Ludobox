package redis

import (
	"context"
	"time"
	"user_service/internal/domain"
	"user_service/internal/repository"
)

func (s *sessionRepository) GetRoomRecommendations(ctx context.Context, key string) (domain.ListResponse[domain.RoomRecommendation], error) {
	exists, err := s.redis.Exists(ctx, key)
	if err != nil {
		return domain.ListResponse[domain.RoomRecommendation]{}, err
	}
	if !exists {
		return domain.ListResponse[domain.RoomRecommendation]{}, repository.ErrorCacheMiss
	}

	var recommendations domain.ListResponse[domain.RoomRecommendation]
	if err := s.redis.Get(ctx, key, &recommendations); err != nil {
		return domain.ListResponse[domain.RoomRecommendation]{}, err
	}

	return recommendations, nil
}

func (s *sessionRepository) SetRoomRecommendations(ctx context.Context, key string, recommendations domain.ListResponse[domain.RoomRecommendation], ttl time.Duration) error {
	return s.redis.Set(ctx, key, recommendations, ttl)
}

func (s *sessionRepository) DeleteKey(ctx context.Context, key string) error {
	return s.redis.Delete(ctx, key)
}

func (s *sessionRepository) DeleteByPrefix(ctx context.Context, prefix string) error {
	return s.redis.DeleteByPrefix(ctx, prefix)
}
