package app

import (
	"user_service/internal/config"
	"user_service/internal/repository"
	"user_service/internal/service"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config      *config.Config
	UserService service.UserService
	Logger      *logrus.Logger
}

func NewApp(
	cfg *config.Config,
	UserRepository repository.UserRepository,
	Logger *logrus.Logger,
) *App {
	baseURL := cfg.PublicURL
	if baseURL == "" {
		baseURL = "http://" + cfg.Domain + ":" + cfg.HttpServerConfig.Port
	}
	UserService := service.NewUserService(UserRepository, Logger)
	return &App{
		Config:      cfg,
		UserService: UserService,
		Logger:      Logger,
	}
}
