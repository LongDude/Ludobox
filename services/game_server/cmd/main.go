package main

import (
	"context"
	"game_server/internal/app"
	"game_server/internal/config"
	"game_server/internal/repository"
	"game_server/internal/repository/postgres"
	"game_server/internal/transport/http"
	"game_server/internal/validation"
	"game_server/pkg/logger"
	"game_server/pkg/storage"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// @title LudoBox API
// @version 1.0
// @description LudoBox GameServer
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
		if err := redisClient.Close(); err != nil {
			appLogger.Warnf("Failed to close redis connection: %v", err)
		}
	}()

	// ! Init HeartBeat
	InternalRepo := postgres.NewInternalRepository(pgPool)
	RoomRepo := postgres.NewRoomRepository(pgPool)
	registration, err := InternalRepo.RegisterGameServer(ctx, repository.GameServerRegistrationParams{
		InstanceKey: cfg.InstanceID,
		RedisHost:   cfg.RedisConfig.Host,
	})
	if err != nil {
		appLogger.Fatalf("Failed to register game server instance with error: %v", err)
		return
	}
	appLogger.WithFields(map[string]interface{}{
		"server_id":    registration.ServerID,
		"instance_key": registration.InstanceKey,
		"redis_host":   registration.RedisHost,
	}).Info("Registered game server instance")

	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	var heartbeatWG sync.WaitGroup
	heartbeatWG.Add(1)
	go func() {
		defer heartbeatWG.Done()

		ticker := time.NewTicker(cfg.HeartbeatInterval.Duration())
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				heartbeatUpdateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := InternalRepo.HeartbeatGameServer(heartbeatUpdateCtx, registration.ServerID)
				cancel()
				if err != nil {
					appLogger.Errorf("Failed to update game server heartbeat: %v", err)
				}
			}
		}
	}()

	usecase := app.NewApp(cfg, InternalRepo, RoomRepo, registration.ServerID, redisClient, appLogger)

	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := usecase.RecoverServerState(recoveryCtx); err != nil {
		recoveryCancel()
		appLogger.Fatalf("Failed to recover game server state: %v", err)
		return
	}
	recoveryCancel()

	// Initialize rooms cache
	initCacheCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := usecase.InitializeCache(initCacheCtx); err != nil {
		initCancel()
		appLogger.Warnf("Failed to initialize rooms cache: %v", err)
	}
	initCancel()
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
	heartbeatCancel()
	heartbeatWG.Wait()

	// ! Graceful shutdown
	err = server.Stop(ctx)
	if err != nil {
		appLogger.Fatal("Server Shutdown:", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := InternalRepo.DeactivateGameServer(shutdownCtx, registration.ServerID); err != nil {
		appLogger.Errorf("Failed to deactivate game server instance: %v", err)
	}
	select {
	case <-ctx.Done():
		appLogger.Info("Timeout stop server")
	default:
		appLogger.Info("Server exiting")
	}

}
