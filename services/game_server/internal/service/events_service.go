package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"game_server/internal/transport/dto"

	"github.com/sirupsen/logrus"
)

// EventsService управляет SSE подписчиками и отправкой событий
type EventsService struct {
	subscribers map[int64][]chan *dto.SSEEvent // roundID -> []chan
	mu          sync.RWMutex
	logger      *logrus.Logger
}

func NewEventsService(logger *logrus.Logger) *EventsService {
	return &EventsService{
		subscribers: make(map[int64][]chan *dto.SSEEvent),
		logger:      logger,
	}
}

// Subscribe подписывает канал на события раунда
func (es *EventsService) Subscribe(roundID int64) chan *dto.SSEEvent {
	es.mu.Lock()
	defer es.mu.Unlock()

	eventChan := make(chan *dto.SSEEvent, 100)
	es.subscribers[roundID] = append(es.subscribers[roundID], eventChan)

	return eventChan
}

// Unsubscribe отписывает канал от событий
func (es *EventsService) Unsubscribe(roundID int64, eventChan chan *dto.SSEEvent) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if chans, exists := es.subscribers[roundID]; exists {
		for i, ch := range chans {
			if ch == eventChan {
				// Удаляем канал из списка
				es.subscribers[roundID] = append(chans[:i], chans[i+1:]...)
				close(eventChan)
				break
			}
		}
		// Если подписчиков нет - удаляем раунд
		if len(es.subscribers[roundID]) == 0 {
			delete(es.subscribers, roundID)
		}
	}
}

// PublishEvent отправляет событие всем подписчикам раунда
func (es *EventsService) PublishEvent(ctx context.Context, roundID int64, eventType string, data interface{}) {
	event := &dto.SSEEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	es.mu.RLock()
	chans := es.subscribers[roundID]
	es.mu.RUnlock()

	for _, ch := range chans {
		select {
		case ch <- event:
		case <-ctx.Done():
			return
		default:
			// Канал переполнен, пропускаем (можно залогировать)
			es.logger.Warnf("Event channel full for round %d", roundID)
		}
	}
}

// PublishPlayerJoined отправляет событие присоединения игрока
func (es *EventsService) PublishPlayerJoined(ctx context.Context, roundID int64, participantID int64, numberInRoom int, currentPlayers int) {
	es.PublishEvent(ctx, roundID, "player_joined", dto.EventPlayerJoined{
		ParticipantID:  participantID,
		NumberInRoom:   numberInRoom,
		CurrentPlayers: currentPlayers,
	})
}

// PublishPlayerLeft отправляет событие выхода игрока
func (es *EventsService) PublishPlayerLeft(ctx context.Context, roundID int64, participantID int64, numberInRoom int, currentPlayers int) {
	es.PublishEvent(ctx, roundID, "player_left", dto.EventPlayerLeft{
		ParticipantID:  participantID,
		NumberInRoom:   numberInRoom,
		CurrentPlayers: currentPlayers,
	})
}

// PublishBoostPurchased отправляет событие покупки буста
func (es *EventsService) PublishBoostPurchased(ctx context.Context, roundID int64, participantID int64, boostPower int) {
	es.PublishEvent(ctx, roundID, "boost_purchased", dto.EventBoostPurchased{
		ParticipantID: participantID,
		BoostPower:    boostPower,
	})
}

// PublishBoostCancelled отправляет событие отмены буста
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

// PublishRoundStarted отправляет событие начала игры
func (es *EventsService) PublishRoundStarted(ctx context.Context, roundID int64, finalPlayers int, gameDurationSec int) {
	es.PublishEvent(ctx, roundID, "round_started", dto.EventRoundStarted{
		RoundID:         roundID,
		FinalPlayers:    finalPlayers,
		GameDurationSec: gameDurationSec,
	})
}

// PublishRoundFinalized отправляет событие завершения игры
func (es *EventsService) PublishRoundFinalized(ctx context.Context, roundID int64, winners []dto.WinnerInfo, payouts map[int64]int64) {
	es.PublishEvent(ctx, roundID, "round_finalized", dto.EventRoundFinalized{
		RoundID: roundID,
		Winners: winners,
		Payouts: payouts,
	})
}

// EncodeSSEMessage кодирует событие в SSE формат
func (es *EventsService) EncodeSSEMessage(event *dto.SSEEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return "data: " + string(data) + "\n\n", nil
}

// GetSubscriberCount возвращает количество подписчиков для раунда (для отладки)
func (es *EventsService) GetSubscriberCount(roundID int64) int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.subscribers[roundID])
}
