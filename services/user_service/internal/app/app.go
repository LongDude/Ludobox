package app

import (
	"user_service/internal/config"
	"user_service/internal/repository"
	"user_service/internal/service"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config        *config.Config
	UserService   service.UserService
	ConfigService service.ConfigService
	Logger        *logrus.Logger
}

func NewApp(
	cfg *config.Config,
	UserRepository repository.UserRepository,
	ConfigRepository repository.ConfigRepository,
	Logger *logrus.Logger,
) *App {
	UserService := service.NewUserService(UserRepository, Logger)
	ConfigService := service.NewConfigService(ConfigRepository, Logger)
	return &App{
		Config:        cfg,
		UserService:   UserService,
		ConfigService: ConfigService,
		Logger:        Logger,
	}
}
