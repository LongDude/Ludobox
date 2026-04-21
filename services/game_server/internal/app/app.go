package app

import (
	"context"
	"game_server/internal/config"
	"game_server/internal/repository"
	"game_server/internal/service"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config          *config.Config
	InternalService service.InternalService
	RoomService     *service.RoomService
	Logger          *logrus.Logger
}

func NewApp(
	cfg *config.Config,
	InternalRepository repository.InternalRepository,
	RoomRepository repository.RoomRepository,
	ServerID int64,
	Logger *logrus.Logger,
) *App {
	baseURL := cfg.PublicURL
	if baseURL == "" {
		baseURL = "http://" + cfg.Domain + ":" + cfg.HttpServerConfig.Port
	}
	InternalService := service.NewInternalService(InternalRepository, Logger)
	RoomService := service.NewRoomService(RoomRepository, Logger, ServerID)
	
	return &App{
		Config:          cfg,
		InternalService: InternalService,
		RoomService:     RoomService,
		Logger:          Logger,
	}
}

// InitializeCache initializes room cache at startup
func (a *App) InitializeCache(ctx context.Context) error {
	return a.RoomService.InitializeRoomsCache(ctx)
}
