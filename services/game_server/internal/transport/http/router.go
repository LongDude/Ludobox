package http

import (
	"user_service/internal/app"
	"user_service/internal/transport/http/handlers"

	"github.com/gin-gonic/gin"
)

func MainRouter(r *gin.RouterGroup, a *app.App) {
	r.GET("/ping", func(ctx *gin.Context) { handlers.PingPong(ctx, a) })
}
func AdminRouter(r *gin.RouterGroup, a *app.App) {
	// admin := r.Group("", middlewares.AdminOnly(a))
}
