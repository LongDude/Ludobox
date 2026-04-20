package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/transport/dto"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

// RecommendRooms godoc
// @Summary Recommend rooms
// @Description Returns recommended open rooms for the authenticated user based on filters and play history.
// @Tags Matchmaking
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param body body dto.RecommendRoomsRequest true "Recommendation filters"
// @Success 200 {object} presenters.RecommendRoomsResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /rooms/recommendations [post]
func RecommendRooms(ctx *gin.Context, a *app.App) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, presenters.ErrorResponse{Error: "missing or invalid X-Authenticated-User"})
		return
	}

	var req dto.RecommendRoomsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.ErrorResponse{Error: err.Error()})
		return
	}

	preferences := domain.MatchmakingPreferences{
		UserID:               userID,
		GameID:               req.GameID,
		MinRegistrationPrice: req.MinRegistrationPrice,
		MaxRegistrationPrice: req.MaxRegistrationPrice,
		MinCapacity:          req.MinCapacity,
		MaxCapacity:          req.MaxCapacity,
		IsBoost:              req.IsBoost,
		MinBoostPower:        req.MinBoostPower,
		Limit:                req.Limit,
		StaleAfter:           a.Config.GameServerStaleAfter.Duration(),
	}

	recommendations, cached, err := a.InternalService.RecommendRooms(ctx.Request.Context(), preferences)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrorInvalidMatchmakingParams):
			ctx.JSON(http.StatusBadRequest, presenters.ErrorResponse{Error: err.Error()})
		default:
			a.Logger.Errorf("recommend rooms failed: %v", err)
			ctx.JSON(http.StatusInternalServerError, presenters.ErrorResponse{Error: "failed to recommend rooms"})
		}
		return
	}

	items := make([]presenters.RoomRecommendationResponse, 0, len(recommendations))
	for _, recommendation := range recommendations {
		items = append(items, toRoomRecommendationResponse(recommendation))
	}

	ctx.JSON(http.StatusOK, presenters.RecommendRoomsResponse{
		Items:  items,
		Cached: cached,
	})
}

// QuickMatch godoc
// @Summary Quick match
// @Description Finds the best available room for the authenticated user and immediately joins it.
// @Tags Matchmaking
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param body body dto.QuickMatchRequest true "Quick match filters"
// @Success 200 {object} presenters.QuickMatchResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /rooms/quick-match [post]
func QuickMatch(ctx *gin.Context, a *app.App) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, presenters.ErrorResponse{Error: "missing or invalid X-Authenticated-User"})
		return
	}

	var req dto.QuickMatchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.ErrorResponse{Error: err.Error()})
		return
	}

	preferences := domain.MatchmakingPreferences{
		UserID:               userID,
		GameID:               req.GameID,
		MinRegistrationPrice: req.MinRegistrationPrice,
		MaxRegistrationPrice: req.MaxRegistrationPrice,
		MinCapacity:          req.MinCapacity,
		MaxCapacity:          req.MaxCapacity,
		IsBoost:              req.IsBoost,
		MinBoostPower:        req.MinBoostPower,
		StaleAfter:           a.Config.GameServerStaleAfter.Duration(),
	}

	result, err := a.InternalService.QuickMatch(ctx.Request.Context(), preferences)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrorInvalidMatchmakingParams):
			ctx.JSON(http.StatusBadRequest, presenters.ErrorResponse{Error: err.Error()})
		case errors.Is(err, domain.ErrorNoAvailableRooms):
			ctx.JSON(http.StatusNotFound, presenters.ErrorResponse{Error: "no available rooms"})
		case errors.Is(err, domain.ErrorUserNotFound):
			ctx.JSON(http.StatusNotFound, presenters.ErrorResponse{Error: "user not found"})
		default:
			a.Logger.Errorf("quick match failed: %v", err)
			ctx.JSON(http.StatusInternalServerError, presenters.ErrorResponse{Error: "failed to select room"})
		}
		return
	}

	ctx.JSON(http.StatusOK, presenters.QuickMatchResponse{
		Room:               toRoomRecommendationResponse(result.RoomRecommendation),
		RoundID:            result.RoundID,
		RoundParticipantID: result.RoundParticipantID,
		SeatNumber:         result.SeatNumber,
		ReusedExistingRoom: result.ReusedExistingRoom,
	})
}

func toRoomRecommendationResponse(recommendation domain.RoomRecommendation) presenters.RoomRecommendationResponse {
	return presenters.RoomRecommendationResponse{
		RoomID:            recommendation.RoomID,
		ConfigID:          recommendation.ConfigID,
		ServerID:          recommendation.ServerID,
		GameID:            recommendation.GameID,
		RegistrationPrice: recommendation.RegistrationPrice,
		Capacity:          recommendation.Capacity,
		MinUsers:          recommendation.MinUsers,
		IsBoost:           recommendation.IsBoost,
		BoostPower:        recommendation.BoostPower,
		CurrentPlayers:    recommendation.CurrentPlayers,
		InstanceKey:       recommendation.InstanceKey,
		RedisHost:         recommendation.RedisHost,
		Score:             recommendation.Score,
	}
}

func authenticatedUserID(ctx *gin.Context) (int64, bool) {
	headerValue := strings.TrimSpace(ctx.GetHeader("X-Authenticated-User"))
	if headerValue == "" {
		return 0, false
	}

	userID, err := strconv.ParseInt(headerValue, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}

	return userID, true
}
