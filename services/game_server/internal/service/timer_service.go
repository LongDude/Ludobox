package service

import (
	"context"
	"sync"
	"time"

	appcfg "game_server/internal/config"
	"game_server/internal/repository"

	"github.com/sirupsen/logrus"
)

type timerState struct {
	cancel   context.CancelFunc
	deadline time.Time
}

// TimerService manages round timers.
type TimerService struct {
	roomRepo      repository.RoomRepository
	eventsService *EventsService
	logger        *logrus.Logger

	timers map[int64]*timerState
	mu     sync.Mutex

	onGameStart    func(ctx context.Context, roundID int64) error
	onGameFinalize func(ctx context.Context, roundID int64) error
	onRoundCancel  func(ctx context.Context, roundID int64) error
}

func NewTimerService(
	roomRepo repository.RoomRepository,
	eventsService *EventsService,
	logger *logrus.Logger,
) *TimerService {
	if logger == nil {
		logger = logrus.New()
	}

	return &TimerService{
		roomRepo:      roomRepo,
		eventsService: eventsService,
		logger:        logger,
		timers:        make(map[int64]*timerState),
	}
}

func (ts *TimerService) SetGameStartCallback(fn func(ctx context.Context, roundID int64) error) {
	ts.onGameStart = fn
}

func (ts *TimerService) SetGameFinalizeCallback(fn func(ctx context.Context, roundID int64) error) {
	ts.onGameFinalize = fn
}

func (ts *TimerService) SetRoundCancelCallback(fn func(ctx context.Context, roundID int64) error) {
	ts.onRoundCancel = fn
}

// StartTimer waits for min_users and then starts the configured round countdown.
func (ts *TimerService) StartTimer(ctx context.Context, roundID int64, roomID int64, minUsers int, configTimeSeconds int) {
	ts.mu.Lock()
	if _, exists := ts.timers[roundID]; exists {
		ts.mu.Unlock()
		ts.logger.Warnf("Timer already started for round %d", roundID)
		return
	}

	timerCtx, cancel := context.WithCancel(ctx)
	ts.timers[roundID] = &timerState{
		cancel:   cancel,
		deadline: time.Now().Add(appcfg.RoundWaitingTimeout),
	}
	ts.mu.Unlock()

	go func() {
		defer ts.clearTimer(roundID)

		ts.logger.Infof("Timer started for round %d, waiting for min_users=%d", roundID, minUsers)

		waitDeadline := time.Now().Add(appcfg.RoundWaitingTimeout)
		ts.setDeadline(roundID, waitDeadline)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

	waitLoop:
		for {
			select {
			case <-timerCtx.Done():
				ts.logger.Infof("Round %d timer cancelled", roundID)
				return
			case <-ticker.C:
				participants, err := ts.roomRepo.GetParticipantsByRoundID(timerCtx, roundID)
				if err != nil {
					ts.logger.Errorf("Error getting participants for round %d: %v", roundID, err)
					continue
				}

				if len(participants) >= minUsers {
					ts.logger.Infof("Round %d reached min_users (%d), starting game timer", roundID, len(participants))
					break waitLoop
				}

				if time.Now().After(waitDeadline) {
					ts.logger.Warnf("Round %d timeout waiting for min_users, cancelling", roundID)
					if ts.onRoundCancel != nil {
						if err := ts.onRoundCancel(timerCtx, roundID); err != nil {
							ts.logger.Errorf("Error cancelling round %d: %v", roundID, err)
						}
					}
					return
				}
			}
		}

		ts.logger.Infof("Starting game for round %d", roundID)
		if ts.onGameStart != nil {
			if err := ts.onGameStart(timerCtx, roundID); err != nil {
				ts.logger.Errorf("Error starting game for round %d: %v", roundID, err)
				return
			}
		}

		participants, _ := ts.roomRepo.GetParticipantsByRoundID(timerCtx, roundID)
		ts.eventsService.PublishRoundStarted(timerCtx, roundID, len(participants), configTimeSeconds)

		gameDeadline := time.Now().Add(time.Duration(configTimeSeconds) * time.Second)
		ts.setDeadline(roundID, gameDeadline)

		gameTimer := time.NewTimer(time.Duration(configTimeSeconds) * time.Second)
		defer gameTimer.Stop()

		select {
		case <-timerCtx.Done():
			ts.logger.Infof("Round %d game timer cancelled", roundID)
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

func (ts *TimerService) StopTimer(roundID int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	state, exists := ts.timers[roundID]
	if !exists {
		return
	}

	state.cancel()
	delete(ts.timers, roundID)
	ts.logger.Infof("Timer stopped for round %d", roundID)
}

func (ts *TimerService) GetRemainingTime(roundID int64) (bool, time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	state, exists := ts.timers[roundID]
	if !exists {
		return false, 0
	}

	remaining := time.Until(state.deadline)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining
}

func (ts *TimerService) setDeadline(roundID int64, deadline time.Time) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if state, exists := ts.timers[roundID]; exists {
		state.deadline = deadline
	}
}

func (ts *TimerService) clearTimer(roundID int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.timers, roundID)
}
