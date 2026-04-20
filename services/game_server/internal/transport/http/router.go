package http

import (
	"game_server/internal/app"
	"game_server/internal/transport/http/handlers"

	"github.com/gin-gonic/gin"
)

func MainRouter(r *gin.RouterGroup, a *app.App) {
	r.GET("/healthz", func(ctx *gin.Context) { handlers.Healthz(ctx, a) })
}
func AdminRouter(r *gin.RouterGroup, a *app.App) {
	// admin := r.Group("", middlewares.AdminOnly(a))
}
