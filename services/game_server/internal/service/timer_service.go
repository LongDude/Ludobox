package service

import (
	"context"
	"math"
	"sync"
	"time"

	appcfg "game_server/internal/config"
	"game_server/internal/repository"

	"github.com/sirupsen/logrus"
)

type timerState struct {
	cancel    context.CancelFunc
	deadline  time.Time
	startedAt *time.Time
	status    string
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

type TimerInfo struct {
	Status    string
	StartedAt *time.Time
	Deadline  *time.Time
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

// StartTimer waits for min_users, then keeps the round in waiting state for the
// configured countdown. Only after that countdown expires does the round become active.
func (ts *TimerService) StartTimer(ctx context.Context, roundID int64, roomID int64, minUsers int, configTimeSeconds int) {
	_ = roomID

	ts.mu.Lock()
	if _, exists := ts.timers[roundID]; exists {
		ts.mu.Unlock()
		ts.logger.Warnf("Timer already started for round %d", roundID)
		return
	}

	timerCtx, cancel := context.WithCancel(ctx)
	ts.timers[roundID] = &timerState{
		cancel: cancel,
		status: "waiting_for_players",
	}
	ts.mu.Unlock()

	go func() {
		defer ts.clearTimer(roundID)

		ts.logger.Infof("Timer started for round %d, waiting for min_users=%d", roundID, minUsers)

		waitDeadline := time.Now().Add(appcfg.RoundWaitingTimeout)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		var countdownDeadline time.Time
		countdownActive := false

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

				now := time.Now()
				if len(participants) < minUsers {
					if countdownActive {
						countdownActive = false
						ts.setPhase(roundID, "waiting_for_players", time.Time{}, nil)
						ts.logger.Infof("Round %d dropped below min_users, resetting countdown", roundID)
					}

					if now.After(waitDeadline) {
						ts.logger.Warnf("Round %d timeout waiting for min_users, cancelling", roundID)
						if ts.onRoundCancel != nil {
							if err := ts.onRoundCancel(timerCtx, roundID); err != nil {
								ts.logger.Errorf("Error cancelling round %d: %v", roundID, err)
							}
						}
						return
					}
					continue
				}

				if !countdownActive {
					countdownActive = true
					startedAt := now
					countdownDeadline = startedAt.Add(time.Duration(configTimeSeconds) * time.Second)
					ts.setPhase(roundID, "waiting", countdownDeadline, &startedAt)
					ts.logger.Infof("Round %d reached min_users (%d), starting waiting countdown", roundID, len(participants))
				}

				secondsLeft := secondsUntil(countdownDeadline)
				ts.eventsService.PublishRoundTimer(timerCtx, roundID, "waiting", secondsLeft)
				if now.Before(countdownDeadline) {
					continue
				}

				ts.logger.Infof("Waiting countdown expired for round %d, starting game", roundID)
				if ts.onGameStart != nil {
					if err := ts.onGameStart(timerCtx, roundID); err != nil {
						ts.logger.Errorf("Error starting game for round %d: %v", roundID, err)
						return
					}
				}

				activeStartedAt := time.Now()
				activeDeadline := activeStartedAt.Add(time.Duration(configTimeSeconds) * time.Second)
				ts.setPhase(roundID, "active", activeDeadline, &activeStartedAt)
				participants, _ = ts.roomRepo.GetParticipantsByRoundID(timerCtx, roundID)
				ts.eventsService.PublishRoundStarted(timerCtx, roundID, len(participants), configTimeSeconds)

				activeTicker := time.NewTicker(1 * time.Second)
				defer activeTicker.Stop()

				for {
					select {
					case <-timerCtx.Done():
						ts.logger.Infof("Round %d active timer cancelled", roundID)
						return
					case <-activeTicker.C:
						secondsLeft = secondsUntil(activeDeadline)
						ts.eventsService.PublishRoundTimer(timerCtx, roundID, "active", secondsLeft)
						if time.Now().Before(activeDeadline) {
							continue
						}

						ts.clearDeadline(roundID, "finished")
						if ts.onGameFinalize != nil {
							if err := ts.onGameFinalize(timerCtx, roundID); err != nil {
								ts.logger.Errorf("Error finalizing round %d: %v", roundID, err)
							}
						}
						return
					}
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
	if !exists || state.deadline.IsZero() {
		return false, 0
	}

	remaining := time.Until(state.deadline)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining
}

func (ts *TimerService) GetTimerInfo(roundID int64) (bool, TimerInfo) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	state, exists := ts.timers[roundID]
	if !exists {
		return false, TimerInfo{}
	}

	info := TimerInfo{Status: state.status}
	if state.startedAt != nil {
		startedAt := *state.startedAt
		info.StartedAt = &startedAt
	}
	if !state.deadline.IsZero() {
		deadline := state.deadline
		info.Deadline = &deadline
	}
	return true, info
}

func (ts *TimerService) setPhase(roundID int64, status string, deadline time.Time, startedAt *time.Time) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if state, exists := ts.timers[roundID]; exists {
		state.status = status
		state.deadline = deadline
		if startedAt == nil {
			state.startedAt = nil
			return
		}
		startedAtCopy := *startedAt
		state.startedAt = &startedAtCopy
	}
}

func (ts *TimerService) clearDeadline(roundID int64, status string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if state, exists := ts.timers[roundID]; exists {
		state.status = status
		state.deadline = time.Time{}
		state.startedAt = nil
	}
}

func (ts *TimerService) clearTimer(roundID int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.timers, roundID)
}

func secondsUntil(deadline time.Time) int {
	if deadline.IsZero() {
		return 0
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0
	}

	return int(math.Ceil(remaining.Seconds()))
}
