package http

import (
	"game_server/internal/app"
	"game_server/internal/transport/http/handlers"

	"github.com/gin-gonic/gin"
)

func MainRouter(r *gin.RouterGroup, a *app.App) {
	r.GET("/healthz", func(ctx *gin.Context) { handlers.Healthz(ctx, a) })

	// Rooms API
	rooms := r.Group("/rooms")
	{
		roomScoped := rooms.Group("/:roomID")
		{
			roomScoped.POST("/join", func(ctx *gin.Context) { handlers.JoinRoom(ctx, a) })
			roomScoped.POST("/join-seat", func(ctx *gin.Context) { handlers.JoinRoomWithSeat(ctx, a) })
			roomScoped.POST("/leave", func(ctx *gin.Context) { handlers.LeaveRoomByUser(ctx, a) })

			participants := roomScoped.Group("/participants/:participantID")
			{
				participants.POST("/boost", func(ctx *gin.Context) { handlers.PurchaseBoost(ctx, a) })
				participants.DELETE("/boost", func(ctx *gin.Context) { handlers.CancelBoost(ctx, a) })
				participants.POST("/leave", func(ctx *gin.Context) { handlers.LeaveRoom(ctx, a) })
			}

			rounds := roomScoped.Group("/rounds")
			{
				rounds.GET("/:roundID", func(ctx *gin.Context) { handlers.GetRoundStatus(ctx, a) })
				rounds.GET("/:roundID/events", func(ctx *gin.Context) { handlers.SubscribeToRoundEvents(ctx, a) })
			}
		}
	}

	// Internal API
	internal := r.Group("/internal")
	{
		internalRounds := internal.Group("/rounds")
		{
			internalRounds.POST("/:roundID/start", func(ctx *gin.Context) { handlers.InternalStartGame(ctx, a) })
			internalRounds.POST("/:roundID/finalize", func(ctx *gin.Context) { handlers.InternalFinalizeGame(ctx, a) })
		}
	}
}
func AdminRouter(r *gin.RouterGroup, a *app.App) {
	// admin := r.Group("", middlewares.AdminOnly(a))
}
