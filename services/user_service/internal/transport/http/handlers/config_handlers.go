package handlers

import (
	"strconv"
	"strings"
	"user_service/internal/app"
	"user_service/internal/domain"

	"github.com/gin-gonic/gin"
)

// Get configs by id
// @Summary Get configs by id
// @Description Get configs by id
// @Tags User
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Success 200 {object} presenters.ConfigResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config/{config_id} [get]
func GetConfigByID(ctx *gin.Context, a *app.App) {

}

// Get configs in use
// @Summary Get configs in use
// @Description Get configs in use with pagination
// @Tags User
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param page query int false "Номер страницы"
// @Param page_size query int false "Количество элементов на странице"
// @Param sort_field query string false "Поле для сортировки"
// @Param sort_direction query string false "Направление сортировки (asc/desc)"
// @Param filter_fields query []string false "Поля для фильтрации"
// @Param filter_operators query []string false "Операторы фильтрации (eq, neq, gt, lt, gte, lte, like, in)"
// @Param filter_values query []string false "Значения для фильтрации"
// @Success 200 {object} presenters.ConfigsResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/configs/used [get]
func GetConfigsInUse(ctx *gin.Context, a *app.App) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	sortField := ctx.Query("sort_field")
	sortDirection := ctx.Query("sort_direction")

	// Get filter parameters as arrays
	filterFields := strings.Split(ctx.Query("filter_fields"), ",")
	filterOperators := strings.Split(ctx.Query("filter_operators"), ",")
	filterValues := strings.Split(ctx.Query("filter_values"), ",")

	params := domain.ListParams{
		Pagination: domain.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
	}

	if sortField != "" && sortDirection != "" {
		params.Sort = &domain.Sort{
			Field:     sortField,
			Direction: sortDirection,
		}
	}

	// Create filters array if we have filter parameters
	if len(filterFields) > 0 && len(filterOperators) > 0 && len(filterValues) > 0 {
		// Ensure all arrays have the same length
		minLen := len(filterFields)
		if len(filterOperators) < minLen {
			minLen = len(filterOperators)
		}
		if len(filterValues) < minLen {
			minLen = len(filterValues)
		}

		params.Filter = make([]domain.Filter, 0, minLen)
		for i := 0; i < minLen; i++ {
			if filterFields[i] != "" && filterOperators[i] != "" && filterValues[i] != "" {
				params.Filter = append(params.Filter, domain.Filter{
					Field:    filterFields[i],
					Operator: filterOperators[i],
					Value:    filterValues[i],
				})
			}
		}
	}

}

// Create new config
// @Summary Create new config
// @Description Create new config
// @Tags User
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Success 200 {object} presenters.ConfigResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config [post]
func CreateNewConfig(ctx *gin.Context, a *app.App) {

}

// Update new config
// @Summary Update new config
// @Description Create new config instead of old config? old config archived for history
// @Tags User
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Success 200 {object} presenters.ConfigResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config/{config_id} [put]
func UpdateConfigByID(ctx *gin.Context, a *app.App) {

}

// Delete config
// @Summary Delete config
// @Description Delete config (archived)
// @Tags User
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Success 204
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config/{config_id} [delete]
func DeleteConfigByID(ctx *gin.Context, a *app.App) {

}
