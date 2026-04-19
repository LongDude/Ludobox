package service

import (
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type ConfigService interface {
}

type configService struct {
	configRepository repository.ConfigRepository
	logger           *logrus.Logger
}

func NewConfigRepository(configRepository repository.ConfigRepository, logger *logrus.Logger) ConfigService {
	return &configService{
		configRepository: configRepository,
		logger:           logger,
	}
}
