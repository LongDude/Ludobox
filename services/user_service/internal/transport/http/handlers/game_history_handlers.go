package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/repository"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

// GetUserGameHistory godoc
// @Summary List current user game history
// @Description Returns current user's game participation history with results and financial outcome.
// @Tags User
// @Accept json
// @Produce json
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param game_id query int false "Game id"
// @Param room_id query int false "Room id"
// @Param status query string false "Result/status: won, lost, left, cancelled, waiting, active"
// @Param date_from query string false "Start date/time, RFC3339 or YYYY-MM-DD"
// @Param date_to query string false "End date/time, RFC3339 or YYYY-MM-DD"
// @Success 200 {object} presenters.GameHistoryResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user/history/games [get]
func GetUserGameHistory(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	params, err := gameHistoryParamsFromQuery(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	history, err := a.GameHistoryService.GetUserGameHistory(ctx, userID, params)
	if err != nil {
		if errors.Is(err, repository.ErrorInvalidListParams) {
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("list game history failed: %w", err)))
		return
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items := make([]presenters.GameHistoryItemResponse, 0, len(history.Items))
	for i := range history.Items {
		items = append(items, gameHistoryToResponse(history.Items[i]))
	}

	ctx.JSON(http.StatusOK, presenters.GameHistoryResponse{
		Items:    items,
		Total:    history.Total,
		Page:     page,
		PageSize: pageSize,
	})
}

func gameHistoryParamsFromQuery(ctx *gin.Context) (domain.GameHistoryParams, error) {
	page, err := optionalPositiveIntQuery(ctx.Query("page"), "page", 1)
	if err != nil {
		return domain.GameHistoryParams{}, err
	}
	pageSize, err := optionalPositiveIntQuery(ctx.Query("page_size"), "page_size", 10)
	if err != nil {
		return domain.GameHistoryParams{}, err
	}

	params := domain.GameHistoryParams{
		Pagination: domain.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
		Status: strings.TrimSpace(ctx.Query("status")),
	}

	if raw := strings.TrimSpace(ctx.Query("game_id")); raw != "" {
		gameID, err := positiveIntQuery(raw, "game_id")
		if err != nil {
			return params, err
		}
		params.GameID = &gameID
	}

	if raw := strings.TrimSpace(ctx.Query("room_id")); raw != "" {
		roomID, err := positiveIntQuery(raw, "room_id")
		if err != nil {
			return params, err
		}
		params.RoomID = &roomID
	}

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

func optionalPositiveIntQuery(raw string, name string, defaultValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	return positiveIntQuery(raw, name)
}

func positiveIntQuery(raw string, name string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func parseHistoryTimeQuery(raw string, endOfDay bool) (time.Time, error) {
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		return value.UTC(), nil
	}

	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		value = value.Add(24*time.Hour - time.Nanosecond)
	}
	return value.UTC(), nil
}

func gameHistoryToResponse(item domain.GameHistoryItem) presenters.GameHistoryItemResponse {
	return presenters.GameHistoryItemResponse{
		RoundID:            item.RoundID,
		RoomID:             item.RoomID,
		GameID:             item.GameID,
		GameName:           item.GameName,
		RoundStatus:        item.RoundStatus,
		Result:             item.Result,
		ReservedSeats:      append([]int(nil), item.ReservedSeats...),
		WinningSeats:       append([]int(nil), item.WinningSeats...),
		ReservedSeatsCount: item.ReservedSeatsCount,
		WinningSeatsCount:  item.WinningSeatsCount,
		EntryFee:           item.EntryFee,
		BoostFee:           item.BoostFee,
		TotalSpent:         item.TotalSpent,
		WinningMoney:       item.WinningMoney,
		NetResult:          item.NetResult,
		JoinedAt:           item.JoinedAt,
		FinishedAt:         item.FinishedAt,
	}
}
