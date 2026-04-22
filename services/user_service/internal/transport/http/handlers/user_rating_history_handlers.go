package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/repository"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

// GetUserRatingHistory godoc
// @Summary List current user rating history
// @Description Returns current user's rating changes for the selected period.
// @Tags User
// @Accept json
// @Produce json
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Param date_from query string false "Start date/time, RFC3339 or YYYY-MM-DD"
// @Param date_to query string false "End date/time, RFC3339 or YYYY-MM-DD"
// @Success 200 {object} presenters.UserRatingHistoryResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user/history/rating [get]
func GetUserRatingHistory(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	params, err := userRatingHistoryParamsFromQuery(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	_, err = a.UserService.GetUserByID(ctx, userID)
	if err != nil {
		if !errors.Is(err, repository.ErrorUserNotFound) {
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("load user failed: %w", err)))
			return
		}
		if _, err := a.UserService.CreateUserByID(ctx, userID); err != nil && !errors.Is(err, repository.ErrorUserAlreadyExist) {
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("auto-create user failed: %w", err)))
			return
		}
	}

	history, err := a.UserService.GetUserRatingHistory(ctx, userID, params)
	if err != nil {
		if errors.Is(err, repository.ErrorUserNotFound) {
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("list rating history failed: %w", err)))
		return
	}

	items := make([]presenters.UserRatingHistoryPointResponse, 0, len(history.Items))
	for i := range history.Items {
		item := history.Items[i]
		items = append(items, presenters.UserRatingHistoryPointResponse{
			HistoryID:   item.HistoryID,
			RoundID:     item.RoundID,
			RoomID:      item.RoomID,
			GameID:      item.GameID,
			GameName:    item.GameName,
			Source:      item.Source,
			Delta:       item.Delta,
			RatingAfter: item.RatingAfter,
			Rank:        item.Rank,
			CreatedAt:   item.CreatedAt,
		})
	}

	ctx.JSON(http.StatusOK, presenters.UserRatingHistoryResponse{
		CurrentRating: history.CurrentRating,
		CurrentRank:   history.CurrentRank,
		PeriodChange:  history.PeriodChange,
		Items:         items,
	})
}

func userRatingHistoryParamsFromQuery(ctx *gin.Context) (domain.UserRatingHistoryParams, error) {
	params := domain.UserRatingHistoryParams{}

	if raw := strings.TrimSpace(ctx.Query("date_from")); raw != "" {
		dateFrom, err := parseHistoryTimeQuery(raw, false)
		if err != nil {
			return params, fmt.Errorf("invalid date_from: %w", err)
		}
		params.DateFrom = &dateFrom
	}

	if raw := strings.TrimSpace(ctx.Query("date_to")); raw != "" {
		dateTo, err := parseHistoryTimeQuery(raw, true)
		if err != nil {
			return params, fmt.Errorf("invalid date_to: %w", err)
		}
		params.DateTo = &dateTo
	}

	if params.DateFrom != nil && params.DateTo != nil && params.DateFrom.After(*params.DateTo) {
		return params, fmt.Errorf("date_from must be before or equal to date_to")
	}

	return params, nil
}
