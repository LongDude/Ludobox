package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/repository"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

func configIDFromPath(ctx *gin.Context) (int, error) {
	raw := strings.TrimSpace(ctx.Param("config_id"))
	if raw == "" {
		return 0, fmt.Errorf("config_id is required")
	}

	configID, err := strconv.Atoi(raw)
	if err != nil || configID <= 0 {
		return 0, fmt.Errorf("config_id must be a positive integer")
	}

	return configID, nil
}

func configToResponse(config *domain.Config) presenters.ConfigResponse {
	if config == nil {
		return presenters.ConfigResponse{}
	}

	return presenters.ConfigResponse{
		ConfigID:            config.ID,
		GameID:              config.GameID,
		Game:                gameToResponse(config.Game),
		Capacity:            config.Capacity,
		RegistrationPrice:   config.RegistrationPrice,
		IsBoost:             config.IsBoost,
		BoostPrice:          config.BoostPrice,
		BoostPower:          config.BoostPower,
		NumberWinners:       config.NumberWinners,
		WinningDistribution: config.WinningDistribution,
		Commission:          config.Commission,
		Time:                config.Time,
		MinUsers:            config.MinUsers,
		ArchivedAt:          config.ArchivedAt,
	}
}

func configRequestToDomain(req presenters.ConfigUpsertRequest) *domain.Config {
	return &domain.Config{
		GameID:              req.GameID,
		Capacity:            req.Capacity,
		RegistrationPrice:   req.RegistrationPrice,
		IsBoost:             req.IsBoost,
		BoostPrice:          req.BoostPrice,
		BoostPower:          req.BoostPower,
		NumberWinners:       req.NumberWinners,
		WinningDistribution: req.WinningDistribution,
		Commission:          req.Commission,
		Time:                req.Time,
		MinUsers:            req.MinUsers,
	}
}

// Get config by id
// @Summary Get config by id
// @Description Returns a single room config by id.
// @Tags Config
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param config_id path int true "Config id"
// @Success 200 {object} presenters.ConfigResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config/{config_id} [get]
func GetConfigByID(ctx *gin.Context, a *app.App) {
	configID, err := configIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	config, err := a.ConfigService.GetConfigByID(ctx, configID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorConfigNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorInvalidListParams):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("get config failed: %w", err)))
		}
		return
	}

	ctx.JSON(http.StatusOK, configToResponse(config))
}

// Get configs in use
// @Summary List active configs
// @Description Returns non-archived configs with pagination, optional sorting and filtering.
// @Tags Config
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
// @Success 200 {object} presenters.ConfigsResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/configs/used [get]
func GetConfigsInUse(ctx *gin.Context, a *app.App) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	sortField := strings.TrimSpace(ctx.Query("sort_field"))
	sortDirection := strings.TrimSpace(ctx.Query("sort_direction"))

	filterFields := strings.Split(strings.TrimSpace(ctx.Query("filter_fields")), ",")
	filterOperators := strings.Split(strings.TrimSpace(ctx.Query("filter_operators")), ",")
	filterValues := strings.Split(strings.TrimSpace(ctx.Query("filter_values")), ",")

	params := domain.ListParams{
		Pagination: domain.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
	}
	if sortField != "" {
		params.Sort = &domain.Sort{
			Field:     sortField,
			Direction: sortDirection,
		}
	}

	minLen := minInt(len(filterFields), len(filterOperators), len(filterValues))
	if minLen > 0 {
		params.Filter = make([]domain.Filter, 0, minLen)
		for i := 0; i < minLen; i++ {
			if filterFields[i] == "" || filterOperators[i] == "" || filterValues[i] == "" {
				continue
			}
			params.Filter = append(params.Filter, domain.Filter{
				Field:    filterFields[i],
				Operator: filterOperators[i],
				Value:    filterValues[i],
			})
		}
	}

	configs, err := a.ConfigService.GetConfigs(ctx, params)
	if err != nil {
		if errors.Is(err, repository.ErrorInvalidListParams) {
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("list configs failed: %w", err)))
		return
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items := make([]presenters.ConfigResponse, 0, len(configs.Items))
	for i := range configs.Items {
		items = append(items, configToResponse(&configs.Items[i]))
	}

	ctx.JSON(http.StatusOK, presenters.ConfigsResponse{
		Items:    items,
		Total:    configs.Total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Create new config
// @Summary Create config
// @Description Creates a new room config.
// @Tags Config
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param request body presenters.ConfigUpsertRequest true "Config payload"
// @Success 201 {object} presenters.ConfigResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config [post]
func CreateNewConfig(ctx *gin.Context, a *app.App) {
	var req presenters.ConfigUpsertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}

	config, err := a.ConfigService.CreateNewConfig(ctx, configRequestToDomain(req))
	if err != nil {
		if errors.Is(err, repository.ErrorInvalidConfig) {
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("create config failed: %w", err)))
		return
	}

	ctx.JSON(http.StatusCreated, configToResponse(config))
}

// Update config by id
// @Summary Replace config by id
// @Description Archives the existing config and creates a new config revision.
// @Tags Config
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param config_id path int true "Config id"
// @Param request body presenters.ConfigUpsertRequest true "Config payload"
// @Success 200 {object} presenters.ConfigResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config/{config_id} [put]
func UpdateConfigByID(ctx *gin.Context, a *app.App) {
	configID, err := configIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	var req presenters.ConfigUpsertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}

	config, err := a.ConfigService.UpdateConfigByID(ctx, configID, configRequestToDomain(req))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorInvalidConfig), errors.Is(err, repository.ErrorInvalidListParams):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		case errors.Is(err, repository.ErrorConfigNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorConfigArchived):
			ctx.JSON(http.StatusConflict, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("update config failed: %w", err)))
		}
		return
	}

	ctx.JSON(http.StatusOK, configToResponse(config))
}

// Delete config
// @Summary Archive config
// @Description Archives an active config.
// @Tags Config
// @Accept json
// @Produce json
// @Param Authorization header string true "Authorization Token"
// @Param config_id path int true "Config id"
// @Success 204
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/config/{config_id} [delete]
func DeleteConfigByID(ctx *gin.Context, a *app.App) {
	configID, err := configIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	err = a.ConfigService.DeleteConfigByID(ctx, configID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorInvalidListParams):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		case errors.Is(err, repository.ErrorConfigNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorConfigArchived):
			ctx.JSON(http.StatusConflict, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("delete config failed: %w", err)))
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

func minInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}

	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}

	return minimum
}
