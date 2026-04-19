package postgres

import (
	"context"
	"user_service/internal/domain"
)

// CreateNewConfig implements [repository.ConfigRepository].
func (c *configRepository) CreateNewConfig(ctx context.Context, config *domain.Config) (*domain.Config, error) {
	panic("unimplemented")
}

// DeleteConfigByID implements [repository.ConfigRepository].
func (c *configRepository) DeleteConfigByID(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetConfigByID implements [repository.ConfigRepository].
func (c *configRepository) GetConfigByID(ctx context.Context, id int) (*domain.Config, error) {
	panic("unimplemented")
}

// GetConfigs implements [repository.ConfigRepository].
func (c *configRepository) GetConfigs(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Config], error) {
	panic("unimplemented")
}

// UpdateConfigByID implements [repository.ConfigRepository].
func (c *configRepository) UpdateConfigByID(ctx context.Context, id int, config *domain.Config) (*domain.Config, error) {
	panic("unimplemented")
}
