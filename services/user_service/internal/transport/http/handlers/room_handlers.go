package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/repository"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

func roomIDFromPath(ctx *gin.Context) (int, error) {
	raw := strings.TrimSpace(ctx.Param("room_id"))
	if raw == "" {
		return 0, fmt.Errorf("room_id is required")
	}

	roomID, err := strconv.Atoi(raw)
	if err != nil || roomID <= 0 {
		return 0, fmt.Errorf("room_id must be a positive integer")
	}

	return roomID, nil
}

func roomToResponse(room *domain.Room) presenters.RoomResponse {
	var config *presenters.ConfigResponse
	if room.Config != nil {
		response := configToResponse(room.Config)
		config = &response
	}

	return presenters.RoomResponse{
		RoomID:     room.ID,
		ConfigID:   room.ConfigID,
		Config:     config,
		ServerID:   room.GameServerID,
		ServerName: room.ServerName,
		Status:     string(room.Status),
		ArchivedAt: room.ArchivedAt,
	}
}

// Get room by id
// @Summary Get room by id
// @Description Returns a single room by id.
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param room_id path int true "Room id"
// @Success 200 {object} presenters.RoomResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room/{room_id} [get]
func GetRoomByID(ctx *gin.Context, a *app.App) {
	roomID, err := roomIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	room, err := a.RoomService.GetRoomByID(ctx, roomID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorRoomNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorInvalidRoom):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("get room failed: %w", err)))
		}
		return
	}

	ctx.JSON(http.StatusOK, roomToResponse(room))
}

// Create room by config
// @Summary Create room by config
// @Description Creates a room and automatically assigns it to the least busy active game server.
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param request body presenters.RoomCreateRequest true "Room create payload"
// @Success 201 {object} presenters.RoomResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room [post]
func CreateRoomByConfigID(ctx *gin.Context, a *app.App) {
	var req presenters.RoomCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}

	room, err := a.RoomService.CreateRoomByConfigID(ctx, req.ConfigID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorInvalidRoom):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		case errors.Is(err, repository.ErrorConfigNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorConfigArchived), errors.Is(err, repository.ErrorNoActiveGameServers):
			ctx.JSON(http.StatusConflict, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("create room failed: %w", err)))
		}
		return
	}

	ctx.JSON(http.StatusCreated, roomToResponse(room))
}

// Update room by id
// @Summary Update room by id
// @Description Updates only archived_at or server_id.
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param room_id path int true "Room id"
// @Param request body presenters.RoomUpdateRequest true "Room update payload"
// @Success 200 {object} presenters.RoomResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room/{room_id} [put]
func UpdateRoomByID(ctx *gin.Context, a *app.App) {
	roomID, err := roomIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	var req presenters.RoomUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}
	if req.ServerID == nil && req.ArchivedAt == nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("server_id or archived_at is required")))
		return
	}

	currentRoom, err := a.RoomService.GetRoomByID(ctx, roomID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorRoomNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorInvalidRoom):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("load room failed: %w", err)))
		}
		return
	}

	update := &domain.Room{
		ID:           currentRoom.ID,
		ConfigID:     currentRoom.ConfigID,
		GameServerID: currentRoom.GameServerID,
		Status:       currentRoom.Status,
		ArchivedAt:   currentRoom.ArchivedAt,
	}
	if req.ServerID != nil {
		update.GameServerID = *req.ServerID
	}
	if req.ArchivedAt != nil {
		update.ArchivedAt = req.ArchivedAt
	}

	room, err := a.RoomService.UpdateRoomByID(ctx, roomID, update)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorInvalidRoom):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		case errors.Is(err, repository.ErrorRoomNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorRoomArchived):
			ctx.JSON(http.StatusConflict, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("update room failed: %w", err)))
		}
		return
	}

	ctx.JSON(http.StatusOK, roomToResponse(room))
}

// Delete room by id
// @Summary Delete room by id
// @Description Archives a room by setting archived_at.
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param room_id path int true "Room id"
// @Success 204
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room/{room_id} [delete]
func DeleteRoomByID(ctx *gin.Context, a *app.App) {
	roomID, err := roomIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	err = a.RoomService.DeleteRoomByID(ctx, roomID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorInvalidRoom):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		case errors.Is(err, repository.ErrorRoomNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorRoomArchived):
			ctx.JSON(http.StatusConflict, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("delete room failed: %w", err)))
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// Get non-archived rooms
// @Summary Returns non-archived rooms
// @Description Returns non-archived rooms. Filter fields: room_id, config_id, server_id, server_name, status, config_capacity, config_registration_price, config_is_boost, config_game_id, config_game_name.
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param sort_field query string false "Sort field"
// @Param sort_direction query string false "Sort direction (asc/desc)"
// @Param filter_fields query []string false "Comma-separated filter fields"
// @Param filter_operators query []string false "Comma-separated filter operators (eq, neq, gt, lt, gte, lte, in, not_in, like, not_like)"
// @Param filter_values query []string false "Comma-separated filter values"
// @Success 200 {object} presenters.RoomsResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/rooms [get]
func GetNotArchivedRooms(ctx *gin.Context, a *app.App) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	sortField := strings.TrimSpace(ctx.Query("sort_field"))
	sortDirection := strings.TrimSpace(ctx.Query("sort_direction"))

	filterFields := strings.Split(strings.TrimSpace(ctx.Query("filter_fields")), ",")
	filterOperators := strings.Split(strings.TrimSpace(ctx.Query("filter_operators")), ",")
	filterValues := strings.Split(strings.TrimSpace(ctx.Query("filter_values")), ",")

	params := domain.ListParams{
		Pagination: domain.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
	}
	if sortField != "" {
		params.Sort = &domain.Sort{
			Field:     sortField,
			Direction: sortDirection,
		}
	}

	minLen := minInt(len(filterFields), len(filterOperators), len(filterValues))
	if minLen > 0 {
		params.Filter = make([]domain.Filter, 0, minLen)
		for i := 0; i < minLen; i++ {
			if filterFields[i] == "" || filterOperators[i] == "" || filterValues[i] == "" {
				continue
			}
			params.Filter = append(params.Filter, domain.Filter{
				Field:    filterFields[i],
				Operator: filterOperators[i],
				Value:    filterValues[i],
			})
		}
	}

	rooms, err := a.RoomService.GetNotArchivedRooms(ctx, params)
	if err != nil {
		if errors.Is(err, repository.ErrorInvalidListParams) {
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("list rooms failed: %w", err)))
		return
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items := make([]presenters.RoomResponse, 0, len(rooms.Items))
	for i := range rooms.Items {
		items = append(items, roomToResponse(&rooms.Items[i]))
	}

	ctx.JSON(http.StatusOK, presenters.RoomsResponse{
		Items:    items,
		Total:    rooms.Total,
		Page:     page,
		PageSize: pageSize,
	})
}
