package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"game_server/internal/repository"
	"game_server/internal/transport/dto"

	"github.com/sirupsen/logrus"
)

const subscriberBufferSize = 256

// EventsService manages SSE subscribers and event delivery.
type EventsService struct {
	subscribers map[int64][]chan *dto.SSEEvent
	mu          sync.RWMutex
	logger      *logrus.Logger
	roomRepo    repository.RoomRepository
}

func NewEventsService(roomRepo repository.RoomRepository, logger *logrus.Logger) *EventsService {
	if logger == nil {
		logger = logrus.New()
	}

	return &EventsService{
		subscribers: make(map[int64][]chan *dto.SSEEvent),
		logger:      logger,
		roomRepo:    roomRepo,
	}
}

// Subscribe registers a buffered channel for round events.
func (es *EventsService) Subscribe(roundID int64) chan *dto.SSEEvent {
	es.mu.Lock()
	defer es.mu.Unlock()

	eventChan := make(chan *dto.SSEEvent, subscriberBufferSize)
	es.subscribers[roundID] = append(es.subscribers[roundID], eventChan)

	return eventChan
}

// Unsubscribe removes the channel from the subscriber list.
func (es *EventsService) Unsubscribe(roundID int64, eventChan chan *dto.SSEEvent) {
	es.mu.Lock()
	defer es.mu.Unlock()

	chans, exists := es.subscribers[roundID]
	if !exists {
		return
	}

	for i, ch := range chans {
		if ch == eventChan {
			es.subscribers[roundID] = append(chans[:i], chans[i+1:]...)
			break
		}
	}

	if len(es.subscribers[roundID]) == 0 {
		delete(es.subscribers, roundID)
	}
}

// PublishEvent broadcasts an event to all subscribers of the round.
func (es *EventsService) PublishEvent(ctx context.Context, roundID int64, eventType string, data interface{}) {
	event := &dto.SSEEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	es.mu.RLock()
	chans := append([]chan *dto.SSEEvent(nil), es.subscribers[roundID]...)
	es.mu.RUnlock()

	for _, ch := range chans {
		if !es.enqueueEvent(ctx, roundID, ch, event) {
			return
		}
	}

	if eventType == "round_timer" || es.roomRepo == nil {
		return
	}

	if err := es.persistEvent(ctx, roundID, eventType, data); err != nil {
		es.logger.WithError(err).Warnf("failed to persist room event %s for round %d", eventType, roundID)
	}
}

func (es *EventsService) PublishPlayerJoined(ctx context.Context, roundID int64, participantID int64, numberInRoom int, currentPlayers int) {
	es.PublishEvent(ctx, roundID, "player_joined", dto.EventPlayerJoined{
		ParticipantID:  participantID,
		NumberInRoom:   numberInRoom,
		CurrentPlayers: currentPlayers,
	})
}

func (es *EventsService) PublishPlayerLeft(ctx context.Context, roundID int64, participantID int64, numberInRoom int, currentPlayers int) {
	es.PublishEvent(ctx, roundID, "player_left", dto.EventPlayerLeft{
		ParticipantID:  participantID,
		NumberInRoom:   numberInRoom,
		CurrentPlayers: currentPlayers,
	})
}

func (es *EventsService) PublishBoostPurchased(ctx context.Context, roundID int64, participantID int64, boostPower int) {
	es.PublishEvent(ctx, roundID, "boost_purchased", dto.EventBoostPurchased{
		ParticipantID: participantID,
		BoostPower:    boostPower,
	})
}

func (es *EventsService) PublishBoostCancelled(ctx context.Context, roundID int64, participantID int64) {
	es.PublishEvent(ctx, roundID, "boost_cancelled", dto.EventBoostCancelled{
		ParticipantID: participantID,
	})
}

func (es *EventsService) PublishRoundTimer(ctx context.Context, roundID int64, status string, secondsLeft int) {
	es.PublishEvent(ctx, roundID, "round_timer", dto.EventRoundTimer{
		RoundID:     roundID,
		Status:      status,
		SecondsLeft: secondsLeft,
	})
}

func (es *EventsService) PublishRoundStarted(ctx context.Context, roundID int64, finalPlayers int, gameDurationSec int) {
	es.PublishEvent(ctx, roundID, "round_started", dto.EventRoundStarted{
		RoundID:         roundID,
		FinalPlayers:    finalPlayers,
		GameDurationSec: gameDurationSec,
	})
}

func (es *EventsService) PublishRoundFinalized(
	ctx context.Context,
	roundID int64,
	winners []dto.WinnerInfo,
	payouts map[int64]int64,
	nextRoundID int64,
	nextRoundDelay int,
) {
	es.PublishEvent(ctx, roundID, "round_finalized", dto.EventRoundFinalized{
		RoundID:        roundID,
		Winners:        winners,
		Payouts:        payouts,
		NextRoundID:    nextRoundID,
		NextRoundDelay: nextRoundDelay,
	})
}

// EncodeSSEMessage marshals the event into SSE wire format.
func (es *EventsService) EncodeSSEMessage(event *dto.SSEEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return "data: " + string(data) + "\n\n", nil
}

func (es *EventsService) GetSubscriberCount(roundID int64) int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.subscribers[roundID])
}

func (es *EventsService) persistEvent(ctx context.Context, roundID int64, eventType string, data interface{}) error {
	roundInfo, err := es.roomRepo.GetRoundInfo(ctx, roundID)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return es.roomRepo.CreateRoomEvent(ctx, roundInfo.RoomID, &roundID, eventType, payload)
}

func (es *EventsService) enqueueEvent(
	ctx context.Context,
	roundID int64,
	ch chan *dto.SSEEvent,
	event *dto.SSEEvent,
) bool {
	select {
	case ch <- event:
		return true
	case <-ctx.Done():
		return false
	default:
	}

	if event.Type == "round_timer" {
		es.logger.Debugf("dropping round_timer for round %d: subscriber channel is full", roundID)
		return true
	}

	select {
	case <-ch:
		es.logger.Warnf("subscriber channel full for round %d, evicting oldest event to deliver %s", roundID, event.Type)
	default:
		es.logger.Warnf("subscriber channel full for round %d, failed to evict oldest event for %s", roundID, event.Type)
		return true
	}

	select {
	case ch <- event:
		return true
	case <-ctx.Done():
		return false
	default:
		es.logger.Warnf("subscriber channel still full for round %d, dropping critical event %s", roundID, event.Type)
		return true
	}
}
