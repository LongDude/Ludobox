package service

import (
	"context"
	"fmt"
	"strings"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type ConfigService interface {
	GetConfigs(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Config], error)
	GetConfigByID(ctx context.Context, id int) (*domain.Config, error)
	CreateNewConfig(ctx context.Context, config *domain.Config) (*domain.Config, error)
	UpdateConfigByID(ctx context.Context, id int, config *domain.Config) (*domain.Config, error)
	DeleteConfigByID(ctx context.Context, id int) error
}

type configService struct {
	configRepository repository.ConfigRepository
	logger           *logrus.Logger
}

func NewConfigService(configRepository repository.ConfigRepository, logger *logrus.Logger) ConfigService {
	return &configService{
		configRepository: configRepository,
		logger:           logger,
	}
}

func (c *configService) GetConfigs(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Config], error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	if params.Sort == nil {
		params.Sort = &domain.Sort{
			Field:     "config_id",
			Direction: "desc",
		}
	} else {
		params.Sort.Field = strings.TrimSpace(params.Sort.Field)
		params.Sort.Direction = strings.ToLower(strings.TrimSpace(params.Sort.Direction))
		if params.Sort.Field == "" {
			return domain.ListResponse[domain.Config]{}, fmt.Errorf("%w: sort field cannot be empty", repository.ErrorInvalidListParams)
		}
		if params.Sort.Direction == "" {
			params.Sort.Direction = "asc"
		}
	}

	return c.configRepository.GetConfigs(ctx, params)
}

func (c *configService) GetConfigByID(ctx context.Context, id int) (*domain.Config, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: config_id must be positive", repository.ErrorInvalidListParams)
	}

	return c.configRepository.GetConfigByID(ctx, id)
}

func (c *configService) CreateNewConfig(ctx context.Context, config *domain.Config) (*domain.Config, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return c.configRepository.CreateNewConfig(ctx, config)
}

func (c *configService) UpdateConfigByID(ctx context.Context, id int, config *domain.Config) (*domain.Config, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: config_id must be positive", repository.ErrorInvalidListParams)
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return c.configRepository.UpdateConfigByID(ctx, id, config)
}

func (c *configService) DeleteConfigByID(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("%w: config_id must be positive", repository.ErrorInvalidListParams)
	}

	return c.configRepository.DeleteConfigByID(ctx, id)
}

func validateConfig(config *domain.Config) error {
	if config == nil {
		return fmt.Errorf("%w: request body is required", repository.ErrorInvalidConfig)
	}
	if config.GameID <= 0 && config.Game != nil {
		config.GameID = config.Game.ID
	}
	if config.GameID <= 0 {
		return fmt.Errorf("%w: game_id must be positive", repository.ErrorInvalidConfig)
	}
	if config.Capacity < 2 || config.Capacity > 20 {
		return fmt.Errorf("%w: capacity must be between 2 and 20", repository.ErrorInvalidConfig)
	}
	if config.RegistrationPrice < 0 {
		return fmt.Errorf("%w: registration_price cannot be negative", repository.ErrorInvalidConfig)
	}
	if config.BoostPrice < 0 {
		return fmt.Errorf("%w: boost_price cannot be negative", repository.ErrorInvalidConfig)
	}
	if config.BoostPower < 0 || config.BoostPower > 100 {
		return fmt.Errorf("%w: boost_power must be between 0 and 100", repository.ErrorInvalidConfig)
	}
	if !config.IsBoost && (config.BoostPrice != 0 || config.BoostPower != 0) {
		return fmt.Errorf("%w: boost_price and boost_power must be zero when is_boost=false", repository.ErrorInvalidConfig)
	}
	if config.NumberWinners < 1 || config.NumberWinners > 20 {
		return fmt.Errorf("%w: number_winners must be between 1 and 20", repository.ErrorInvalidConfig)
	}
	if config.NumberWinners > config.Capacity {
		return fmt.Errorf("%w: number_winners cannot exceed capacity", repository.ErrorInvalidConfig)
	}
	if len(config.WinningDistribution) != config.NumberWinners {
		return fmt.Errorf("%w: winning_distribution length must match number_winners", repository.ErrorInvalidConfig)
	}
	distributionSum := 0
	for _, value := range config.WinningDistribution {
		if value < 0 || value > 100 {
			return fmt.Errorf("%w: winning_distribution values must be between 0 and 100", repository.ErrorInvalidConfig)
		}
		distributionSum += value
	}
	if distributionSum != 100 {
		return fmt.Errorf("%w: winning_distribution must sum to 100", repository.ErrorInvalidConfig)
	}
	if config.Commission < 0 || config.Commission > 100 {
		return fmt.Errorf("%w: commission must be between 0 and 100", repository.ErrorInvalidConfig)
	}
	if config.Time <= 0 {
		return fmt.Errorf("%w: time must be greater than 0", repository.ErrorInvalidConfig)
	}
	if config.MinUsers < 1 {
		return fmt.Errorf("%w: min_users must be at least 1", repository.ErrorInvalidConfig)
	}
	if config.MinUsers > config.Capacity {
		return fmt.Errorf("%w: min_users cannot exceed capacity", repository.ErrorInvalidConfig)
	}

	return nil
}
