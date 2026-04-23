package main

import (
	"context"
	"errors"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user_service/internal/app"
	"user_service/internal/config"
	"user_service/internal/repository/postgres"
	"user_service/internal/service"
	transporthttp "user_service/internal/transport/http"
	"user_service/internal/validation"
	"user_service/pkg/logger"
	"user_service/pkg/storage"
)

// @title LudoBox API
// @version 1.0
// @description LudoBox UserService
// @host localhost:8080
// @BasePath /api
func main() {
	// Init validator
	validation.Init()
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()
	// ! Init logger
	appLogger := logger.LoggerSetup("info")
	// ! Parse config from env
	cfg, err := config.MustLoadConfig()
	if err != nil {
		appLogger.Fatalf("Failed to load config with error: %v", err)
		return
	}
	appLogger = logger.LoggerSetup(cfg.LogLevel)
	// ! Init repoisitory
	// ! Init postgres
	pgPool, err := storage.PostgresConnect(startupCtx, cfg.PostgresConfig)
	if err != nil {
		appLogger.Fatalf("Failed to create pool conection to postgres with error: %v", err)
		return
	}
	defer pgPool.Close()
	// ! Init redis
	// redisClient, err := storage.RedisConnect(ctx, cfg.RedisConfig)
	// if err != nil {
	// 	logger.Fatalf("Failed to create conection to redis with error: %v", err)
	// 	return
	// }

	// SessionRepo := redis.NewSessionRepository(redisClient)
	UserRepo := postgres.NewUserRepository(pgPool)
	GameHistoryRepo := postgres.NewGameHistoryRepository(pgPool)
	GameRepo := postgres.NewGameRepository(pgPool)
	ConfigRepo := postgres.NewConfigRepository(pgPool)
	RoomRepo := postgres.NewRoomRepository(pgPool)
	GameServerRepo := postgres.NewGameServerRepository(pgPool)

	usecase := app.NewApp(cfg, UserRepo, GameHistoryRepo, GameRepo, ConfigRepo, RoomRepo, GameServerRepo, appLogger)
	adminEvents := service.NewAdminEventService(pgPool, appLogger)
	usecase.AdminEvents = adminEvents
	adminEvents.Start(context.Background())
	defer adminEvents.Stop()
	userBalanceEvents := service.NewUserBalanceEventService(pgPool, appLogger)
	usecase.UserBalanceEvents = userBalanceEvents
	userBalanceEvents.Start(context.Background())
	defer userBalanceEvents.Stop()

	// ! Init REST
	server := transporthttp.NewHTTPServer(cfg, usecase)
	appLogger.Info("Start HTTP server")
	go func() {
		listenErr := server.Listen()
		if listenErr != nil && !errors.Is(listenErr, nethttp.ErrServerClosed) {
			appLogger.Fatalf("Failed to start HTTP server: %v", listenErr)
		}
	}()
	//Wait for interrupt signal to shutdown server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutdown HTTP Server ...")

	// ! Graceful shutdown
	adminEvents.Stop()
	userBalanceEvents.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	err = server.Stop(shutdownCtx)
	if err != nil {
		appLogger.Fatal("Server Shutdown:", err)
	}
	select {
	case <-shutdownCtx.Done():
		appLogger.Info("Timeout stop server")
	default:
		appLogger.Info("Server exiting")
	}

}
