package http

import (
	"user_service/internal/app"
	"user_service/internal/transport/http/handlers"

	"github.com/gin-gonic/gin"
)

func MainRouter(r *gin.RouterGroup, a *app.App) {
	r.GET("/healthz", func(ctx *gin.Context) { handlers.Healthz(ctx, a) })
	r.POST("/rooms/recommendations", func(ctx *gin.Context) { handlers.RecommendRooms(ctx, a) })
	r.POST("/rooms/quick-match", func(ctx *gin.Context) { handlers.QuickMatch(ctx, a) })
}

func InternalRouter(r *gin.RouterGroup, a *app.App) {
	r.GET("/rooms/:room_id/owner", func(ctx *gin.Context) { handlers.ResolveRoomOwner(ctx, a) })
	r.GET("/game-servers/next", func(ctx *gin.Context) { handlers.SelectAvailableGameServer(ctx, a) })
}

func AdminRouter(r *gin.RouterGroup, a *app.App) {
	// admin := r.Group("", middlewares.AdminOnly(a))
}
