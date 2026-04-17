package handlers

import (
	"net/http"
	"user_service/internal/app"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

// ping pong
// @Summary ping pong
// @Description ping pong
// @Tags Test
// @Accept json
// @Produce json
// @Success 200 {object} presenters.TestResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /ping [get]
func PingPong(ctx *gin.Context, a *app.App) {
	ctx.JSON(http.StatusOK, presenters.TestResponse{
		Pong: "pong",
	})
}
