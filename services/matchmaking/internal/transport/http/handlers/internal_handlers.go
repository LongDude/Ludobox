package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

func ResolveRoomOwner(ctx *gin.Context, a *app.App) {
	roomID, err := strconv.ParseInt(ctx.Param("room_id"), 10, 64)
	if err != nil || roomID <= 0 {
		ctx.JSON(http.StatusBadRequest, presenters.ErrorResponse{
			Error: "invalid room_id",
		})
		return
	}

	gameServer, err := a.InternalService.ResolveRoomOwner(ctx.Request.Context(), roomID, a.Config.GameServerStaleAfter.Duration())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrorRoomNotFound):
			ctx.JSON(http.StatusNotFound, presenters.ErrorResponse{
				Error: "room owner not found",
			})
		case errors.Is(err, domain.ErrorGameServerUnavailable):
			ctx.JSON(http.StatusServiceUnavailable, presenters.ErrorResponse{
				Error: "room owner is unavailable",
			})
		default:
			a.Logger.Errorf("resolve room owner failed: %v", err)
			ctx.JSON(http.StatusInternalServerError, presenters.ErrorResponse{
				Error: "failed to resolve room owner",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, presenters.ResolveServerResponse{
		Server_id:    gameServer.ServerID,
		Instance_key: gameServer.InstanceKey,
		Redis_host:   gameServer.RedisHost,
	})
}

func SelectAvailableGameServer(ctx *gin.Context, a *app.App) {
	gameServer, err := a.InternalService.SelectAvailableGameServer(ctx.Request.Context(), a.Config.GameServerStaleAfter.Duration())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrorNoActiveGameServers):
			ctx.JSON(http.StatusServiceUnavailable, presenters.ErrorResponse{
				Error: "no active game servers",
			})
		default:
			a.Logger.Errorf("select available game server failed: %v", err)
			ctx.JSON(http.StatusInternalServerError, presenters.ErrorResponse{
				Error: "failed to select game server",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, presenters.ServerResponse{
		Server_id:      gameServer.ServerID,
		Instance_key:   gameServer.InstanceKey,
		Redis_host:     gameServer.RedisHost,
		Active_rooms:   gameServer.ActiveRooms,
		Last_heartbeat: gameServer.LastHeartbeatAt,
	})
}
