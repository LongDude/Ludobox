package main

import (
	"authorization_service/internal/app"
	"authorization_service/internal/config"
	"authorization_service/internal/repository/postgres"
	"authorization_service/internal/repository/redis"
	"authorization_service/internal/transport/http"
	"authorization_service/internal/transport/rpc"
	"authorization_service/internal/validation"
	"authorization_service/pkg/logger"
	"authorization_service/pkg/storage"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// @title SSO API
// @version 1.0
// @description SSO для регистрации и авторизации сервисов
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

	SessionRepo := redis.NewSessionRepository(redisClient)
	UserRepo := postgres.NewUserRepository(pgPool)
	TokenBlock := redis.NewTokenBlocklist(redisClient)

	usecase := app.NewApp(cfg, SessionRepo, UserRepo, TokenBlock, appLogger)
	// ! Init REST
	server := http.NewHTTPServer(cfg, usecase)
	appLogger.Info("Start HTTP server")
	go func() {
		err = server.Listen()
		if err != nil {
			appLogger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()
	grpcServer := rpc.New(appLogger, usecase, cfg.GRPCConfig.Port)
	appLogger.Info("Start gRPC server")
	go func() {
		if err := grpcServer.Run(); err != nil {
			appLogger.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()
	//Wait for interrupt signal to shutdown server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutdown HTTP Server ...")

	// Stop gRPC server
	grpcServer.Stop()

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
