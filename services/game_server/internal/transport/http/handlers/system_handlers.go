package handlers

import (
	"net/http"
	"game_server/internal/app"

	"github.com/gin-gonic/gin"
)

// Healthz godoc
// @Summary Service health
// @Description Lightweight liveness probe
// @Tags System
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func Healthz(ctx *gin.Context, a *app.App) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
