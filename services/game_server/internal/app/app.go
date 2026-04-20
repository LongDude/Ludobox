package app

import (
	"game_server/internal/config"
	"game_server/internal/repository"
	"game_server/internal/service"

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
	Logger *logrus.Logger,
) *App {
	baseURL := cfg.PublicURL
	if baseURL == "" {
		baseURL = "http://" + cfg.Domain + ":" + cfg.HttpServerConfig.Port
	}
	InternalService := service.NewInternalService(InternalRepository, Logger)
	return &App{
		Config:          cfg,
		InternalService: InternalService,
		Logger:          Logger,
	}
}
