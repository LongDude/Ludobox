package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

const userBalanceEventsChannel = "user_balance_events"

type UserBalanceEvent struct {
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	UserID    int64     `json:"user_id"`
	Balance   int64     `json:"balance"`
	Timestamp time.Time `json:"timestamp"`
}

type UserBalanceEvents interface {
	Start(ctx context.Context)
	Stop()
	Done() <-chan struct{}
	Subscribe(ctx context.Context, userID int64) (<-chan UserBalanceEvent, func())
}

type UserBalanceEventService struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger

	mu          sync.RWMutex
	subscribers map[int64]map[chan UserBalanceEvent]struct{}

	startOnce sync.Once
	stopOnce  sync.Once
	cancel    context.CancelFunc
	done      chan struct{}
	wg        sync.WaitGroup
}

func NewUserBalanceEventService(pool *pgxpool.Pool, logger *logrus.Logger) *UserBalanceEventService {
	return &UserBalanceEventService{
		pool:        pool,
		logger:      logger,
		subscribers: make(map[int64]map[chan UserBalanceEvent]struct{}),
		done:        make(chan struct{}),
	}
}

func (s *UserBalanceEventService) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		listenCtx, cancel := context.WithCancel(ctx)
		s.cancel = cancel

		if s.pool == nil {
			s.logWarn("user balance events listener not started: postgres pool is nil")
			return
		}

		s.wg.Add(1)
		go s.listenLoop(listenCtx)
	})
}

func (s *UserBalanceEventService) Stop() {
	s.stopOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		close(s.done)
		s.wg.Wait()
	})
}

func (s *UserBalanceEventService) Done() <-chan struct{} {
	return s.done
}

func (s *UserBalanceEventService) Subscribe(ctx context.Context, userID int64) (<-chan UserBalanceEvent, func()) {
	ch := make(chan UserBalanceEvent, 16)

	select {
	case <-s.done:
		close(ch)
		return ch, func() {}
	default:
	}

	s.mu.Lock()
	if s.subscribers[userID] == nil {
		s.subscribers[userID] = make(map[chan UserBalanceEvent]struct{})
	}
	s.subscribers[userID][ch] = struct{}{}
	s.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subscribers[userID], ch)
			if len(s.subscribers[userID]) == 0 {
				delete(s.subscribers, userID)
			}
			s.mu.Unlock()
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			unsubscribe()
		case <-s.done:
			unsubscribe()
		}
	}()

	return ch, unsubscribe
}

func (s *UserBalanceEventService) Publish(event UserBalanceEvent) {
	select {
	case <-s.done:
		return
	default:
	}

	s.mu.RLock()
	targets := make([]chan UserBalanceEvent, 0, len(s.subscribers[event.UserID]))
	for subscriber := range s.subscribers[event.UserID] {
		targets = append(targets, subscriber)
	}
	s.mu.RUnlock()

	for _, target := range targets {
		select {
		case target <- event:
		default:
			select {
			case <-target:
			default:
			}
			select {
			case target <- event:
			default:
			}
		}
	}
}

func (s *UserBalanceEventService) listenLoop(ctx context.Context) {
	defer s.wg.Done()

	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}

		err := s.listenOnce(ctx)
		if err == nil || ctx.Err() != nil {
			return
		}

		s.logWarnf("user balance events listener disconnected: %v", err)

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (s *UserBalanceEventService) listenOnce(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire postgres listener connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN "+userBalanceEventsChannel); err != nil {
		return fmt.Errorf("listen %s: %w", userBalanceEventsChannel, err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "UNLISTEN "+userBalanceEventsChannel)
	}()

	s.logInfo("user balance events listener started")

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("wait for postgres notification: %w", err)
		}

		event, err := parseUserBalanceEventNotification(notification)
		if err != nil {
			s.logWarnf("skip malformed user balance notification: %v", err)
			continue
		}

		s.Publish(event)
	}
}

func parseUserBalanceEventNotification(notification *pgconn.Notification) (UserBalanceEvent, error) {
	if notification == nil {
		return UserBalanceEvent{}, fmt.Errorf("notification is nil")
	}

	var event UserBalanceEvent
	if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
		return UserBalanceEvent{}, fmt.Errorf("decode payload: %w", err)
	}

	if event.Type == "" {
		event.Type = "user_balance_changed"
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	return event, nil
}

func (s *UserBalanceEventService) logInfo(message string) {
	if s.logger != nil {
		s.logger.Info(message)
	}
}

func (s *UserBalanceEventService) logWarn(message string) {
	if s.logger != nil {
		s.logger.Warn(message)
	}
}

func (s *UserBalanceEventService) logWarnf(format string, args ...any) {
	if s.logger != nil {
		s.logger.Warnf(format, args...)
	}
}
