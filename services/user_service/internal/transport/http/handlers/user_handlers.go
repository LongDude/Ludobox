package handlers

import (
	"net/http"
	"user_service/internal/app"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

// Get user by id
// @Summary user by id
// @Description get user data by id
// @Headers X-Authenticated-User required
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} presenters.UserResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [get]
func GetUserByID(ctx *gin.Context, a *app.App) {
	ctx.JSON(http.StatusNotImplemented, presenters.UserResponse{})
	// If user doesn't exist, create by user_id with nickname "user_+user_id"
}

// Create user by id
// @Summary user by id
// @Description Create user by id
// @Headers X-Authenticated-User required
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} presenters.UserResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [post]
func CreateUserByID(ctx *gin.Context, a *app.App) {
	ctx.JSON(http.StatusNotImplemented, presenters.UserResponse{})
}

// Update user by id
// @Summary user by id
// @Description Update user by id
// @Headers X-Authenticated-User required
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} presenters.UserResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [put]
func UpdateUserByID(ctx *gin.Context, a *app.App) {
	ctx.JSON(http.StatusNotImplemented, presenters.UserResponse{})
}

// Delete user by id
// @Summary user by id
// @Description Delete user by id
// @Headers X-Authenticated-User required
// @Tags User
// @Accept json
// @Produce json
// @Success 200
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [delete]
func DeleteUserByID(ctx *gin.Context, a *app.App) {
	ctx.Status(http.StatusNotImplemented)
}

// Update user balance by id
// @Summary user by id
// @Description Update user balance by id
// @Headers X-Authenticated-User required
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} presenters.UserResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user/balance [put]
func UpdateUserBalanceByID(ctx *gin.Context, a *app.App) {
	ctx.JSON(http.StatusNotImplemented, presenters.UserResponse{})
}
