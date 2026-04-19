package http

import (
	"user_service/internal/app"
	"user_service/internal/transport/http/handlers"
	"user_service/internal/transport/http/middlewares"

	"github.com/gin-gonic/gin"
)

func MainRouter(r *gin.RouterGroup, a *app.App) {
	r.GET("/healthz", func(ctx *gin.Context) { handlers.Healthz(ctx, a) })
	r.GET("/ping", func(ctx *gin.Context) { handlers.PingPong(ctx, a) })
	r.GET("/user", func(ctx *gin.Context) { handlers.GetUserByID(ctx, a) })
	r.POST("/user", func(ctx *gin.Context) { handlers.CreateUserByID(ctx, a) })
	r.PUT("/user", func(ctx *gin.Context) { handlers.UpdateUserByID(ctx, a) })
	r.DELETE("/user", func(ctx *gin.Context) { handlers.DeleteUserByID(ctx, a) })
	r.PUT("/user/balance", func(ctx *gin.Context) { handlers.UpdateUserBalanceByID(ctx, a) })
}
func AdminRouter(r *gin.RouterGroup, a *app.App) {
	admin := r.Group("", middlewares.AdminOnly(a))
	admin.GET("/configs/used", func(ctx *gin.Context) { handlers.GetConfigsInUse(ctx, a) })
	admin.GET("/config/:config_id", func(ctx *gin.Context) { handlers.GetConfigByID(ctx, a) })
	admin.POST("/config", func(ctx *gin.Context) { handlers.CreateNewConfig(ctx, a) })
	admin.PUT("/config/:config_id", func(ctx *gin.Context) { handlers.UpdateConfigByID(ctx, a) })
	admin.DELETE("/config/:config_id", func(ctx *gin.Context) { handlers.DeleteConfigByID(ctx, a) })
}
