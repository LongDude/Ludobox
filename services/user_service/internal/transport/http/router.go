package http

import (
	"user_service/internal/app"
	"user_service/internal/transport/http/handlers"
	"user_service/internal/transport/http/middlewares"

	"github.com/gin-gonic/gin"
)

func MainRouter(r *gin.RouterGroup, a *app.App) {
	r.GET("/healthz", func(ctx *gin.Context) { handlers.Healthz(ctx, a) })
	r.GET("/user", func(ctx *gin.Context) { handlers.GetUserByID(ctx, a) })
	r.POST("/user", func(ctx *gin.Context) { handlers.CreateUserByID(ctx, a) })
	r.PUT("/user", func(ctx *gin.Context) { handlers.UpdateUserByID(ctx, a) })
	r.DELETE("/user", func(ctx *gin.Context) { handlers.DeleteUserByID(ctx, a) })
	r.PUT("/user/balance", func(ctx *gin.Context) { handlers.UpdateUserBalanceByID(ctx, a) })
	r.GET("/user/balance/events", func(ctx *gin.Context) { handlers.SubscribeUserBalanceEvents(ctx, a) })
	r.GET("/user/history/games", func(ctx *gin.Context) { handlers.GetUserGameHistory(ctx, a) })
	r.GET("/user/history/rating", func(ctx *gin.Context) { handlers.GetUserRatingHistory(ctx, a) })
}
func AdminRouter(r *gin.RouterGroup, a *app.App) {
	admin := r.Group("", middlewares.AdminOnly(a))
	admin.GET("/games", func(ctx *gin.Context) { handlers.GetGames(ctx, a) })
	admin.GET("/game/:game_id", func(ctx *gin.Context) { handlers.GetGameByID(ctx, a) })
	admin.POST("/game", func(ctx *gin.Context) { handlers.CreateGame(ctx, a) })
	admin.PUT("/game/:game_id", func(ctx *gin.Context) { handlers.UpdateGameByID(ctx, a) })
	admin.DELETE("/game/:game_id", func(ctx *gin.Context) { handlers.DeleteGameByID(ctx, a) })
	admin.GET("/configs/used", func(ctx *gin.Context) { handlers.GetConfigsInUse(ctx, a) })
	admin.GET("/config/:config_id", func(ctx *gin.Context) { handlers.GetConfigByID(ctx, a) })
	admin.POST("/config", func(ctx *gin.Context) { handlers.CreateNewConfig(ctx, a) })
	admin.PUT("/config/:config_id", func(ctx *gin.Context) { handlers.UpdateConfigByID(ctx, a) })
	admin.DELETE("/config/:config_id", func(ctx *gin.Context) { handlers.DeleteConfigByID(ctx, a) })
	admin.GET("/rooms", func(ctx *gin.Context) { handlers.GetNotArchivedRooms(ctx, a) })
	admin.GET("/room/:room_id", func(ctx *gin.Context) { handlers.GetRoomByID(ctx, a) })
	admin.POST("/room", func(ctx *gin.Context) { handlers.CreateRoomByConfigID(ctx, a) })
	admin.PUT("/room/:room_id", func(ctx *gin.Context) { handlers.UpdateRoomByID(ctx, a) })
	admin.DELETE("/room/:room_id", func(ctx *gin.Context) { handlers.DeleteRoomByID(ctx, a) })
	admin.GET("/servers", func(ctx *gin.Context) { handlers.GetServers(ctx, a) })
	admin.GET("/events", func(ctx *gin.Context) { handlers.SubscribeAdminEvents(ctx, a) })
}
