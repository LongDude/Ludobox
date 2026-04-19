package handlers

import (
	"user_service/internal/app"

	"github.com/gin-gonic/gin"
)

// Get room by id
// @Summary Get room by id
// @Description Returns a single room room by id.
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param room_id path int true "Room id"
// @Success 200 {object} presenters.RoomResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room/{room_id} [get]
func GetRoomByID(ctx *gin.Context, a *app.App) {

}

// Get room by id
// @Summary Create room by config
// @Description Returns a single room. Automaticly add room to less busy server in this moment
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param body presenters.RoomCreateReq
// @Success 200 {object} presenters.RoomResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room [post]
func CreateRoomByConfigID(ctx *gin.Context, a *app.App) {
}

// Get room by id
// @Summary Update room by id
// @Description Update room by id, update only archived_at or server_id
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param room_id path int true "Room id"
// @Success 200 {object} presenters.RoomResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room/{room_id} [put]
func UpdateRoomByID(ctx *gin.Context, a *app.App) {
}

// Get room by id
// @Summary Delete room by id
// @Description Update ArchiveAt in Room
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param room_id path int true "Room id"
// @Success 204
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/room/{room_id} [delete]
func DeleteRoomByID(ctx *gin.Context, a *app.App) {
}

// Get room by id
// @Summary Returns non-archived rooms
// @Description Returns non-archived rooms.
// @Tags Room
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param sort_field query string false "Sort field"
// @Param sort_direction query string false "Sort direction (asc/desc)"
// @Param filter_fields query []string false "Comma-separated filter fields"
// @Param filter_operators query []string false "Comma-separated filter operators (eq, neq, gt, lt, gte, lte, in, not_in, contains, contained, overlap)"
// @Param filter_values query []string false "Comma-separated filter values"
// @Success 200 {object} presenters.RoomsResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/rooms [get]
func GetNotArchivedRooms(ctx *gin.Context, a *app.App) {

}
