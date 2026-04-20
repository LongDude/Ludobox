package app

import (
	"user_service/internal/config"
	"user_service/internal/repository"
	"user_service/internal/service"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config            *config.Config
	UserService       service.UserService
	GameService       service.GameService
	ConfigService     service.ConfigService
	RoomService       service.RoomService
	GameServerService service.GameServerService
	Logger            *logrus.Logger
}

func NewApp(
	cfg *config.Config,
	UserRepository repository.UserRepository,
	GameRepository repository.GameRepository,
	ConfigRepository repository.ConfigRepository,
	RoomRepository repository.RoomRepository,
	GameServerRepository repository.GameServerRepository,
	Logger *logrus.Logger,
) *App {
	UserService := service.NewUserService(UserRepository, Logger)
	GameService := service.NewGameService(GameRepository, Logger)
	ConfigService := service.NewConfigService(ConfigRepository, Logger)
	RoomService := service.NewRoomService(RoomRepository, ConfigRepository, cfg, Logger)
	GameServerService := service.NewGameServerService(GameServerRepository, Logger)
	return &App{
		Config:            cfg,
		UserService:       UserService,
		GameService:       GameService,
		ConfigService:     ConfigService,
		RoomService:       RoomService,
		GameServerService: GameServerService,
		Logger:            Logger,
	}
}
