package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"user_service/internal/domain"
	"user_service/internal/repository"
)

const (
	defaultRecommendationLimit int32         = 10
	maxRecommendationLimit     int32         = 50
	defaultQuickMatchLimit     int32         = 5
	defaultStaleAfter          time.Duration = 15 * time.Second
)

func (s *internalService) RecommendRooms(ctx context.Context, preferences domain.MatchmakingPreferences) ([]domain.RoomRecommendation, bool, error) {
	normalizedPreferences, err := normalizeMatchmakingPreferences(preferences, defaultRecommendationLimit)
	if err != nil {
		return nil, false, err
	}

	return s.recommendRooms(ctx, normalizedPreferences, true)
}

func (s *internalService) QuickMatch(ctx context.Context, preferences domain.MatchmakingPreferences) (*domain.QuickMatchResult, error) {
	normalizedPreferences, err := normalizeMatchmakingPreferences(preferences, defaultQuickMatchLimit)
	if err != nil {
		return nil, err
	}
	if normalizedPreferences.Limit < defaultQuickMatchLimit {
		normalizedPreferences.Limit = defaultQuickMatchLimit
	}

	existingMembership, err := s.internalRepository.GetUserActiveRoom(ctx, normalizedPreferences.UserID)
	if err == nil {
		return &domain.QuickMatchResult{
			RoomMembership:     *existingMembership,
			ReusedExistingRoom: true,
		}, nil
	}
	if !errors.Is(err, domain.ErrorActiveRoomNotFound) {
		return nil, err
	}

	recommendations, _, err := s.recommendRooms(ctx, normalizedPreferences, false)
	if err != nil {
		return nil, err
	}
	if len(recommendations) == 0 {
		return nil, domain.ErrorNoAvailableRooms
	}

	for _, recommendation := range recommendations {
		membership, joinErr := s.internalRepository.JoinRoom(ctx, normalizedPreferences.UserID, recommendation.RoomID)
		if joinErr == nil {
			return &domain.QuickMatchResult{
				RoomMembership:     *membership,
				ReusedExistingRoom: false,
			}, nil
		}

		if errors.Is(joinErr, domain.ErrorRoomFull) || errors.Is(joinErr, domain.ErrorRoomUnavailable) {
			continue
		}

		if errors.Is(joinErr, domain.ErrorUserAlreadyInRoom) {
			activeMembership, activeErr := s.internalRepository.GetUserActiveRoom(ctx, normalizedPreferences.UserID)
			if activeErr != nil {
				return nil, joinErr
			}
			return &domain.QuickMatchResult{
				RoomMembership:     *activeMembership,
				ReusedExistingRoom: true,
			}, nil
		}

		return nil, joinErr
	}

	return nil, domain.ErrorNoAvailableRooms
}

func (s *internalService) recommendRooms(
	ctx context.Context,
	preferences domain.MatchmakingPreferences,
	useCache bool,
) ([]domain.RoomRecommendation, bool, error) {
	if useCache && s.sessionRepository != nil && s.recommendationTTL > 0 {
		cacheKey := recommendationCacheKey(preferences)
		recommendations, err := s.sessionRepository.GetRoomRecommendations(ctx, cacheKey)
		if err == nil {
			return recommendations, true, nil
		}
		if !errors.Is(err, repository.ErrorCacheMiss) {
			s.logger.Warnf("failed to get recommendations from cache: %v", err)
		}

		recommendations, repoErr := s.internalRepository.RecommendRooms(ctx, preferences)
		if repoErr != nil {
			return nil, false, repoErr
		}

		if setErr := s.sessionRepository.SetRoomRecommendations(ctx, cacheKey, recommendations, s.recommendationTTL); setErr != nil {
			s.logger.Warnf("failed to set recommendations cache: %v", setErr)
		}

		return recommendations, false, nil
	}

	recommendations, err := s.internalRepository.RecommendRooms(ctx, preferences)
	if err != nil {
		return nil, false, err
	}

	return recommendations, false, nil
}

func normalizeMatchmakingPreferences(
	preferences domain.MatchmakingPreferences,
	defaultLimit int32,
) (domain.MatchmakingPreferences, error) {
	if preferences.UserID <= 0 {
		return domain.MatchmakingPreferences{}, fmt.Errorf("%w: user_id must be positive", domain.ErrorInvalidMatchmakingParams)
	}

	if preferences.MinRegistrationPrice != nil && *preferences.MinRegistrationPrice < 0 {
		return domain.MatchmakingPreferences{}, fmt.Errorf("%w: min_registration_price must be non-negative", domain.ErrorInvalidMatchmakingParams)
	}
	if preferences.MaxRegistrationPrice != nil && *preferences.MaxRegistrationPrice < 0 {
		return domain.MatchmakingPreferences{}, fmt.Errorf("%w: max_registration_price must be non-negative", domain.ErrorInvalidMatchmakingParams)
	}
	if preferences.MinRegistrationPrice != nil && preferences.MaxRegistrationPrice != nil &&
		*preferences.MinRegistrationPrice > *preferences.MaxRegistrationPrice {
		return domain.MatchmakingPreferences{}, fmt.Errorf("%w: min_registration_price cannot be greater than max_registration_price", domain.ErrorInvalidMatchmakingParams)
	}

	if preferences.MinCapacity != nil && *preferences.MinCapacity <= 0 {
		return domain.MatchmakingPreferences{}, fmt.Errorf("%w: min_capacity must be positive", domain.ErrorInvalidMatchmakingParams)
	}
	if preferences.MaxCapacity != nil && *preferences.MaxCapacity <= 0 {
		return domain.MatchmakingPreferences{}, fmt.Errorf("%w: max_capacity must be positive", domain.ErrorInvalidMatchmakingParams)
	}
	if preferences.MinCapacity != nil && preferences.MaxCapacity != nil &&
		*preferences.MinCapacity > *preferences.MaxCapacity {
		return domain.MatchmakingPreferences{}, fmt.Errorf("%w: min_capacity cannot be greater than max_capacity", domain.ErrorInvalidMatchmakingParams)
	}

	if preferences.MinBoostPower != nil {
		if *preferences.MinBoostPower < 0 || *preferences.MinBoostPower > 100 {
			return domain.MatchmakingPreferences{}, fmt.Errorf("%w: min_boost_power must be in range [0,100]", domain.ErrorInvalidMatchmakingParams)
		}
	}

	if preferences.Limit <= 0 {
		preferences.Limit = defaultLimit
	}
	if preferences.Limit > maxRecommendationLimit {
		preferences.Limit = maxRecommendationLimit
	}

	if preferences.StaleAfter <= 0 {
		preferences.StaleAfter = defaultStaleAfter
	}

	return preferences, nil
}

func recommendationCacheKey(preferences domain.MatchmakingPreferences) string {
	parts := []string{
		"matchmaking:recommend:v1",
		"user:" + strconv.FormatInt(preferences.UserID, 10),
		"game:" + optionalInt64ToString(preferences.GameID),
		"price_min:" + optionalInt64ToString(preferences.MinRegistrationPrice),
		"price_max:" + optionalInt64ToString(preferences.MaxRegistrationPrice),
		"capacity_min:" + optionalInt32ToString(preferences.MinCapacity),
		"capacity_max:" + optionalInt32ToString(preferences.MaxCapacity),
		"is_boost:" + optionalBoolToString(preferences.IsBoost),
		"min_boost_power:" + optionalInt32ToString(preferences.MinBoostPower),
		"limit:" + strconv.FormatInt(int64(preferences.Limit), 10),
	}

	return strings.Join(parts, "|")
}

func optionalInt64ToString(value *int64) string {
	if value == nil {
		return "any"
	}
	return strconv.FormatInt(*value, 10)
}

func optionalInt32ToString(value *int32) string {
	if value == nil {
		return "any"
	}
	return strconv.FormatInt(int64(*value), 10)
}

func optionalBoolToString(value *bool) string {
	if value == nil {
		return "any"
	}
	if *value {
		return "true"
	}
	return "false"
}
