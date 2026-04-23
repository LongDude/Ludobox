package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user_service/internal/app"
	"user_service/internal/config"
	"user_service/internal/repository/postgres"
	"user_service/internal/repository/redis"
	"user_service/internal/transport/http"
	"user_service/internal/validation"
	"user_service/pkg/logger"
	"user_service/pkg/storage"
)

// @title LudoBox API
// @version 1.0
// @description LudoBox Matchmaking
// @host localhost:8080
// @BasePath /api
func main() {
	// Init validator
	validation.Init()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
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
	pgPool, err := storage.PostgresConnect(ctx, cfg.PostgresConfig)
	if err != nil {
		appLogger.Fatalf("Failed to create pool conection to postgres with error: %v", err)
		return
	}
	// ! Init redis
	redisClient, err := storage.RedisConnect(ctx, cfg.RedisConfig)
	if err != nil {
		appLogger.Fatalf("Failed to create conection to redis with error: %v", err)
		return
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			appLogger.Errorf("Failed to close redis with error: %v", closeErr)
		}
	}()

	SessionRepo := redis.NewSessionRepository(redisClient)
	InternalRepo := postgres.NewInternalRepository(pgPool)

	usecase := app.NewApp(cfg, InternalRepo, SessionRepo, appLogger)
	// ! Init REST
	server := http.NewHTTPServer(cfg, usecase)
	appLogger.Info("Start HTTP server")
	go func() {
		err = server.Listen()
		if err != nil {
			appLogger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()
	//Wait for interrupt signal to shutdown server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutdown HTTP Server ...")

	// ! Graceful shutdown
	err = server.Stop(ctx)
	if err != nil {
		appLogger.Fatal("Server Shutdown:", err)
	}
	select {
	case <-ctx.Done():
		appLogger.Info("Timeout stop server")
	default:
		appLogger.Info("Server exiting")
	}

}
