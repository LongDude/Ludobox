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

const adminEventsChannel = "admin_user_service_events"

type AdminEvent struct {
	Type      string    `json:"type"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

type AdminEvents interface {
	Start(ctx context.Context)
	Stop()
	Done() <-chan struct{}
	Subscribe(ctx context.Context) (<-chan AdminEvent, func())
}

type AdminEventService struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger

	mu          sync.RWMutex
	subscribers map[chan AdminEvent]struct{}

	startOnce sync.Once
	stopOnce  sync.Once
	cancel    context.CancelFunc
	done      chan struct{}
	wg        sync.WaitGroup
}

func NewAdminEventService(pool *pgxpool.Pool, logger *logrus.Logger) *AdminEventService {
	return &AdminEventService{
		pool:        pool,
		logger:      logger,
		subscribers: make(map[chan AdminEvent]struct{}),
		done:        make(chan struct{}),
	}
}

func (s *AdminEventService) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		listenCtx, cancel := context.WithCancel(ctx)
		s.cancel = cancel

		if s.pool == nil {
			s.logWarn("admin events listener not started: postgres pool is nil")
			return
		}

		s.wg.Add(1)
		go s.listenLoop(listenCtx)
	})
}

func (s *AdminEventService) Stop() {
	s.stopOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		close(s.done)
		s.wg.Wait()
	})
}

func (s *AdminEventService) Done() <-chan struct{} {
	return s.done
}

func (s *AdminEventService) Subscribe(ctx context.Context) (<-chan AdminEvent, func()) {
	ch := make(chan AdminEvent, 32)

	select {
	case <-s.done:
		close(ch)
		return ch, func() {}
	default:
	}

	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subscribers, ch)
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

func (s *AdminEventService) Publish(event AdminEvent) {
	select {
	case <-s.done:
		return
	default:
	}

	s.mu.RLock()
	subscribers := make([]chan AdminEvent, 0, len(s.subscribers))
	for subscriber := range s.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	s.mu.RUnlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- event:
			default:
			}
		}
	}
}

func (s *AdminEventService) listenLoop(ctx context.Context) {
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

		s.logWarnf("admin events listener disconnected: %v", err)

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

func (s *AdminEventService) listenOnce(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire postgres listener connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN "+adminEventsChannel); err != nil {
		return fmt.Errorf("listen %s: %w", adminEventsChannel, err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "UNLISTEN "+adminEventsChannel)
	}()

	s.logInfo("admin events listener started")

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("wait for postgres notification: %w", err)
		}

		event, err := parseAdminEventNotification(notification)
		if err != nil {
			s.logWarnf("skip malformed admin event notification: %v", err)
			continue
		}

		s.Publish(event)
	}
}

func parseAdminEventNotification(notification *pgconn.Notification) (AdminEvent, error) {
	if notification == nil {
		return AdminEvent{}, fmt.Errorf("notification is nil")
	}

	var event AdminEvent
	if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
		return AdminEvent{}, fmt.Errorf("decode payload: %w", err)
	}

	if event.Type == "" {
		event.Type = "admin_resource_changed"
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	return event, nil
}

func (s *AdminEventService) logInfo(message string) {
	if s.logger != nil {
		s.logger.Info(message)
	}
}

func (s *AdminEventService) logWarn(message string) {
	if s.logger != nil {
		s.logger.Warn(message)
	}
}

func (s *AdminEventService) logWarnf(format string, args ...any) {
	if s.logger != nil {
		s.logger.Warnf(format, args...)
	}
}
