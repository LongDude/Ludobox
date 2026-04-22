package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

const adminEventsChannel = "admin_user_service_events"

type AdminEvent struct {
	Type      string          `json:"type"`
	Resource  string          `json:"resource"`
	Action    string          `json:"action"`
	ID        int64           `json:"id"`
	Data      json.RawMessage `json:"data,omitempty" swaggertype:"object"`
	Timestamp time.Time       `json:"timestamp"`
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

		s.enrichAdminEvent(ctx, &event)
		s.Publish(event)
	}
}

func (s *AdminEventService) enrichAdminEvent(ctx context.Context, event *AdminEvent) {
	if event == nil || s.pool == nil || event.ID <= 0 || event.Action == "delete" {
		return
	}

	data, err := s.loadAdminEventData(ctx, *event)
	if err != nil {
		s.logWarnf("skip admin event payload enrichment for %s/%d: %v", event.Resource, event.ID, err)
		return
	}
	if len(data) == 0 {
		return
	}

	event.Data = data
}

func (s *AdminEventService) loadAdminEventData(ctx context.Context, event AdminEvent) (json.RawMessage, error) {
	switch event.Resource {
	case "servers":
		return s.loadServerEventData(ctx, event.ID)
	case "rooms":
		return s.loadRoomEventData(ctx, event.ID)
	case "games":
		return s.loadGameEventData(ctx, event.ID)
	case "configs":
		return s.loadConfigEventData(ctx, event.ID)
	default:
		return nil, nil
	}
}

func (s *AdminEventService) loadServerEventData(ctx context.Context, id int64) (json.RawMessage, error) {
	const query = `
		SELECT server_id, instance_key, redis_host, status, started_at, last_heartbeat_at, archived_at
		FROM game_servers
		WHERE server_id = $1
	`

	var (
		serverID        int64
		instanceKey     string
		redisHost       string
		status          string
		startedAt       sql.NullTime
		lastHeartbeatAt sql.NullTime
		archivedAt      sql.NullTime
	)
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&serverID,
		&instanceKey,
		&redisHost,
		&status,
		&startedAt,
		&lastHeartbeatAt,
		&archivedAt,
	)
	if err != nil {
		return nil, ignoreMissingAdminEventRow(err)
	}

	payload := map[string]any{
		"server_id":    serverID,
		"instance_key": instanceKey,
		"redis_host":   redisHost,
		"status":       status,
	}
	putTime(payload, "started_at", startedAt)
	putTime(payload, "last_heartbeat_at", lastHeartbeatAt)
	putTime(payload, "archived_at", archivedAt)
	return marshalAdminEventData(payload)
}

func (s *AdminEventService) loadRoomEventData(ctx context.Context, id int64) (json.RawMessage, error) {
	const query = `
		SELECT r.room_id, r.config_id, r.server_id, r.current_players, r.status, r.archived_at, gs.instance_key
		FROM rooms r
		LEFT JOIN game_servers gs ON gs.server_id = r.server_id
		WHERE r.room_id = $1
	`

	var (
		roomID         int64
		configID       int64
		serverID       int64
		currentPlayers int32
		status         string
		archivedAt     sql.NullTime
		serverName     sql.NullString
	)
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&roomID,
		&configID,
		&serverID,
		&currentPlayers,
		&status,
		&archivedAt,
		&serverName,
	)
	if err != nil {
		return nil, ignoreMissingAdminEventRow(err)
	}

	payload := map[string]any{
		"room_id":         roomID,
		"config_id":       configID,
		"server_id":       serverID,
		"current_players": currentPlayers,
		"status":          status,
	}
	putTime(payload, "archived_at", archivedAt)
	if serverName.Valid {
		payload["server_name"] = serverName.String
	}
	return marshalAdminEventData(payload)
}

func (s *AdminEventService) loadGameEventData(ctx context.Context, id int64) (json.RawMessage, error) {
	const query = `
		SELECT game_id, name_game, archived_at
		FROM games
		WHERE game_id = $1
	`

	var (
		gameID     int64
		name       string
		archivedAt sql.NullTime
	)
	err := s.pool.QueryRow(ctx, query, id).Scan(&gameID, &name, &archivedAt)
	if err != nil {
		return nil, ignoreMissingAdminEventRow(err)
	}

	payload := map[string]any{
		"game_id":   gameID,
		"name_game": name,
	}
	putTime(payload, "archived_at", archivedAt)
	return marshalAdminEventData(payload)
}

func (s *AdminEventService) loadConfigEventData(ctx context.Context, id int64) (json.RawMessage, error) {
	const query = `
		SELECT
			config_id, game_id, capacity, registration_price, is_boost, boost_price, boost_power,
			number_winners, winning_distribution, commission, time, round_time, next_round_delay,
			min_users, archived_at
		FROM config
		WHERE config_id = $1
	`

	var (
		configID            int64
		gameID              int64
		capacity            int32
		registrationPrice   int64
		isBoost             bool
		boostPrice          int64
		boostPower          int32
		numberWinners       int32
		winningDistribution []int32
		commission          int32
		waitingTime         int32
		roundTime           int32
		nextRoundDelay      int32
		minUsers            int32
		archivedAt          sql.NullTime
	)
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&configID,
		&gameID,
		&capacity,
		&registrationPrice,
		&isBoost,
		&boostPrice,
		&boostPower,
		&numberWinners,
		&winningDistribution,
		&commission,
		&waitingTime,
		&roundTime,
		&nextRoundDelay,
		&minUsers,
		&archivedAt,
	)
	if err != nil {
		return nil, ignoreMissingAdminEventRow(err)
	}

	payload := map[string]any{
		"config_id":            configID,
		"game_id":              gameID,
		"capacity":             capacity,
		"registration_price":   registrationPrice,
		"is_boost":             isBoost,
		"boost_price":          boostPrice,
		"boost_power":          boostPower,
		"number_winners":       numberWinners,
		"winning_distribution": winningDistribution,
		"commission":           commission,
		"time":                 waitingTime,
		"round_time":           roundTime,
		"next_round_delay":     nextRoundDelay,
		"min_users":            minUsers,
	}
	putTime(payload, "archived_at", archivedAt)
	return marshalAdminEventData(payload)
}

func parseAdminEventNotification(notification *pgconn.Notification) (AdminEvent, error) {
	if notification == nil {
		return AdminEvent{}, fmt.Errorf("notification is nil")
	}

	payload := strings.TrimSpace(notification.Payload)
	if payload == "" {
		return AdminEvent{}, fmt.Errorf("notification payload is empty")
	}

	if strings.HasPrefix(payload, "{") {
		return parseAdminEventJSONPayload(payload)
	}

	return parseAdminEventTextPayload(payload)
}

func parseAdminEventJSONPayload(payload string) (AdminEvent, error) {
	var event AdminEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return AdminEvent{}, fmt.Errorf("decode payload: %w", err)
	}

	return normalizeAdminEvent(event), nil
}

func parseAdminEventTextPayload(payload string) (AdminEvent, error) {
	parts := strings.Split(payload, "|")
	if len(parts) != 4 {
		return AdminEvent{}, fmt.Errorf("decode text payload: expected 4 fields, got %d", len(parts))
	}

	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return AdminEvent{}, fmt.Errorf("decode text payload id: %w", err)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[3])
	if err != nil {
		return AdminEvent{}, fmt.Errorf("decode text payload timestamp: %w", err)
	}

	return normalizeAdminEvent(AdminEvent{
		Type:      "admin_resource_changed",
		Resource:  parts[0],
		Action:    parts[1],
		ID:        id,
		Timestamp: timestamp.UTC(),
	}), nil
}

func normalizeAdminEvent(event AdminEvent) AdminEvent {
	if event.Type == "" {
		event.Type = "admin_resource_changed"
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	return event
}

func putTime(payload map[string]any, key string, value sql.NullTime) {
	if value.Valid {
		payload[key] = value.Time.UTC()
	}
}

func marshalAdminEventData(payload map[string]any) (json.RawMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func ignoreMissingAdminEventRow(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
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
