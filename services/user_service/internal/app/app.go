package app

import (
	"user_service/internal/config"
	"user_service/internal/repository"
	"user_service/internal/service"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config       *config.Config
	AdminService service.AdminService
	Logger       *logrus.Logger
}

func NewApp(
	cfg *config.Config,
	AdminRepository repository.AdminRepository,
	Logger *logrus.Logger,
) *App {
	baseURL := cfg.PublicURL
	if baseURL == "" {
		baseURL = "http://" + cfg.Domain + ":" + cfg.HttpServerConfig.Port
	}
	AdminService := service.NewAdminService(AdminRepository, Logger)
	return &App{
		Config:       cfg,
		AdminService: AdminService,
		Logger:       Logger,
	}
}
