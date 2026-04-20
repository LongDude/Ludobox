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

func gameServerToResponse(server *domain.GameServer) presenters.GameServerResponse {
	return presenters.GameServerResponse{
		ServerID:        server.ServerID,
		InstanceKey:     server.InstanceKey,
		RedisHost:       server.RedisHost,
		Status:          server.Status,
		StartedAt:       server.StartedAt,
		LastHeartbeatAt: server.LastHeartbeatAt,
		ArchivedAt:      server.ArchivedAt,
	}
}

// GetServers godoc
// @Summary List game servers
// @Description Returns all game servers with pagination and sorting.
// @Tags Server
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param sort_field query string false "Sort field"
// @Param sort_direction query string false "Sort direction (asc/desc)"
// @Success 200 {object} presenters.GameServersResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/servers [get]
func GetServers(ctx *gin.Context, a *app.App) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	sortField := strings.TrimSpace(ctx.Query("sort_field"))
	sortDirection := strings.TrimSpace(ctx.Query("sort_direction"))

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

	servers, err := a.GameServerService.GetServers(ctx, params)
	if err != nil {
		if errors.Is(err, repository.ErrorInvalidListParams) {
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("list game servers failed: %w", err)))
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

	items := make([]presenters.GameServerResponse, 0, len(servers.Items))
	for i := range servers.Items {
		items = append(items, gameServerToResponse(&servers.Items[i]))
	}

	ctx.JSON(http.StatusOK, presenters.GameServersResponse{
		Items:    items,
		Total:    servers.Total,
		Page:     page,
		PageSize: pageSize,
	})
}
