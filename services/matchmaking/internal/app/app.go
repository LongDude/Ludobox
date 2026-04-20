package app

import (
	"user_service/internal/config"
	"user_service/internal/repository"
	"user_service/internal/service"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config          *config.Config
	InternalService service.InternalService
	Logger          *logrus.Logger
}

func NewApp(
	cfg *config.Config,
	InternalRepository repository.InternalRepository,
	SessionRepository repository.SessionRepository,
	Logger *logrus.Logger,
) *App {
	InternalService := service.NewInternalService(
		InternalRepository,
		SessionRepository,
		cfg.RecommendationCacheTTL.Duration(),
		Logger,
	)
	return &App{
		Config:          cfg,
		InternalService: InternalService,
		Logger:          Logger,
	}
}
