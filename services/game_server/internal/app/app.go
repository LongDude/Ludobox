package app

import (
	"context"
	"game_server/internal/config"
	"game_server/internal/repository"
	"game_server/internal/service"
	"game_server/internal/transport/dto"
	"game_server/pkg/storage"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config          *config.Config
	InternalService service.InternalService
	RoomService     *service.RoomService
	EventsService   *service.EventsService
	TimerService    *service.TimerService
	Logger          *logrus.Logger
}

func NewApp(
	cfg *config.Config,
	InternalRepository repository.InternalRepository,
	RoomRepository repository.RoomRepository,
	ServerID int64,
	RoomCache *storage.RedisClient,
	Logger *logrus.Logger,
) *App {
	InternalService := service.NewInternalService(InternalRepository, Logger)
	EventsService := service.NewEventsService(RoomRepository, Logger)
	RoomService := service.NewRoomService(RoomRepository, Logger, ServerID, RoomCache, cfg.RNGServiceURL)
	TimerService := service.NewTimerService(RoomRepository, EventsService, Logger)
	RoomService.SetTimerService(TimerService)
	TimerService.SetGameStartCallback(RoomService.StartGameRound)
	TimerService.SetRoundCancelCallback(RoomService.CancelWaitingRound)
	TimerService.SetGameFinalizeCallback(func(ctx context.Context, roundID int64) error {
		winners, err := RoomService.FinalizeGameRound(ctx, roundID)
		if err != nil {
			return err
		}

		winnerInfos := make([]dto.WinnerInfo, 0, len(winners))
		payouts := make(map[int64]int64, len(winners))
		for _, winner := range winners {
			var userID *int64
			if !winner.IsBot && winner.UserID > 0 {
				value := winner.UserID
				userID = &value
			}
			winnerInfos = append(winnerInfos, dto.WinnerInfo{
				ParticipantID: winner.RoundParticipantID,
				UserID:        userID,
				Nickname:      winner.NickName,
				NumberInRoom:  winner.NumberInRoom,
				Winnings:      winner.WinningMoney,
				GrossWinnings: winner.GrossWinningMoney,
				IsBot:         winner.IsBot,
			})
			payouts[winner.RoundParticipantID] = winner.WinningMoney
		}

		nextRoundID := int64(0)
		nextRoundDelay := 0
		roomInfo, roomErr := RoomService.GetRoomInfoByRound(ctx, roundID)
		if roomErr == nil && roomInfo != nil {
			if roomInfo.CurrentRoundID != nil {
				nextRoundID = *roomInfo.CurrentRoundID
			}
			if roomInfo.Config != nil {
				nextRoundDelay = roomInfo.Config.NextRoundDelay
			}
		}

		EventsService.PublishRoundFinalized(ctx, roundID, winnerInfos, payouts, nextRoundID, nextRoundDelay)
		return nil
	})

	return &App{
		Config:          cfg,
		InternalService: InternalService,
		RoomService:     RoomService,
		EventsService:   EventsService,
		TimerService:    TimerService,
		Logger:          Logger,
	}
}

// InitializeCache initializes room cache at startup
func (a *App) InitializeCache(ctx context.Context) error {
	return a.RoomService.InitializeRoomsCache(ctx)
}

func (a *App) RecoverServerState(ctx context.Context) error {
	return a.RoomService.RecoverServerState(ctx)
}
