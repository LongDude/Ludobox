package service

import (
	"context"
	"sync"
	"time"

	appcfg "game_server/internal/config"
	"game_server/internal/repository"

	"github.com/sirupsen/logrus"
)

// TimerService управляет таймерами раундов
type TimerService struct {
	roomRepo      repository.RoomRepository
	eventsService *EventsService
	logger        *logrus.Logger

	// Отслеживание активных таймеров
	timers map[int64]context.CancelFunc
	mu     sync.Mutex

	// Callback для запуска игры
	onGameStart func(ctx context.Context, roundID int64) error
	// Callback для завершения игры
	onGameFinalize func(ctx context.Context, roundID int64) error
}

func NewTimerService(
	roomRepo repository.RoomRepository,
	eventsService *EventsService,
	logger *logrus.Logger,
) *TimerService {
	return &TimerService{
		roomRepo:      roomRepo,
		eventsService: eventsService,
		logger:        logger,
		timers:        make(map[int64]context.CancelFunc),
	}
}

// SetGameStartCallback устанавливает callback для запуска игры
func (ts *TimerService) SetGameStartCallback(fn func(ctx context.Context, roundID int64) error) {
	ts.onGameStart = fn
}

// SetGameFinalizeCallback устанавливает callback для завершения игры
func (ts *TimerService) SetGameFinalizeCallback(fn func(ctx context.Context, roundID int64) error) {
	ts.onGameFinalize = fn
}

// StartTimer запускает таймер для раунда
// Сначала ждет min_users, потом запускает таймер игры на configTime сек
func (ts *TimerService) StartTimer(ctx context.Context, roundID int64, roomID int64, minUsers int, configTimeSeconds int) {
	ts.mu.Lock()
	if _, exists := ts.timers[roundID]; exists {
		ts.mu.Unlock()
		ts.logger.Warnf("Timer already started for round %d", roundID)
		return
	}

	timerCtx, cancel := context.WithCancel(ctx)
	ts.timers[roundID] = cancel
	ts.mu.Unlock()

	go func() {
		defer func() {
			ts.mu.Lock()
			delete(ts.timers, roundID)
			ts.mu.Unlock()
		}()

		// Фаза 1: ждём min_users за 15 минут
		ts.logger.Infof("Timer started for round %d, waiting for min_users=%d", roundID, minUsers)

		startWaitTime := time.Now()
		waitDeadline := startWaitTime.Add(appcfg.RoundWaitingTimeout)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		minUsersReached := false

	waitLoop:
		for {
			select {
			case <-timerCtx.Done():
				ts.logger.Infof("Round %d timer cancelled", roundID)
				return
			case <-ticker.C:
				// Проверяем количество игроков
				participants, err := ts.roomRepo.GetParticipantsByRoundID(timerCtx, roundID)
				if err != nil {
					ts.logger.Errorf("Error getting participants for round %d: %v", roundID, err)
					continue
				}

				activeCount := len(participants)
				if activeCount >= minUsers && !minUsersReached {
					minUsersReached = true
					ts.logger.Infof("Round %d reached min_users (%d), starting game timer", roundID, activeCount)
					break waitLoop
				}

				// Проверяем timeout
				if time.Now().After(waitDeadline) {
					ts.logger.Warnf("Round %d timeout waiting for min_users, cancelling", roundID)
					if err := ts.cancelRound(timerCtx, roundID); err != nil {
						ts.logger.Errorf("Error cancelling round %d: %v", roundID, err)
					}
					return
				}
			}
		}

		// Фаза 2: стартуем игру
		ts.logger.Infof("Starting game for round %d", roundID)
		if ts.onGameStart != nil {
			if err := ts.onGameStart(timerCtx, roundID); err != nil {
				ts.logger.Errorf("Error starting game for round %d: %v", roundID, err)
				return
			}
		}

		// Отправляем событие
		participants, _ := ts.roomRepo.GetParticipantsByRoundID(timerCtx, roundID)
		ts.eventsService.PublishRoundStarted(timerCtx, roundID, len(participants), configTimeSeconds)

		// Фаза 3: ждём configTime секунд
		gameTimer := time.NewTimer(time.Duration(configTimeSeconds) * time.Second)
		defer gameTimer.Stop()

		select {
		case <-timerCtx.Done():
			ts.logger.Infof("Round %d game timer cancelled", roundID)
			gameTimer.Stop()
			return
		case <-gameTimer.C:
			ts.logger.Infof("Game timer expired for round %d, finalizing", roundID)
			if ts.onGameFinalize != nil {
				if err := ts.onGameFinalize(timerCtx, roundID); err != nil {
					ts.logger.Errorf("Error finalizing round %d: %v", roundID, err)
					return
				}
			}
		}
	}()
}

// StopTimer останавливает таймер раунда
func (ts *TimerService) StopTimer(roundID int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if cancel, exists := ts.timers[roundID]; exists {
		cancel()
		delete(ts.timers, roundID)
		ts.logger.Infof("Timer stopped for round %d", roundID)
	}
}

// cancelRound отменяет раунд (мягкое удаление) и выкидывает всех пользователей
func (ts *TimerService) cancelRound(ctx context.Context, roundID int64) error {
	// TODO: Реализовать отмену раунда и возврат денег всем участникам
	// На данный момент это должно быть сделано через RoomService.LeaveRoom для каждого
	return nil
}

// GetRemainingTime возвращает оставшееся время до завершения (для отладки)
func (ts *TimerService) GetRemainingTime(roundID int64) (bool, time.Duration) {
	ts.mu.Lock()
	_, exists := ts.timers[roundID]
	ts.mu.Unlock()

	if !exists {
		return false, 0
	}
	return true, 0 // TODO: реализовать отслеживание времени
}
