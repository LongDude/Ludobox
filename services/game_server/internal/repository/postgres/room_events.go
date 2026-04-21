package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"game_server/internal/domain"
)

func (r *roomRepo) CreateRoomEvent(ctx context.Context, roomID int64, roundID *int64, eventType string, eventData json.RawMessage) error {
	if len(eventData) == 0 {
		eventData = json.RawMessage(`{}`)
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO room_events (room_id, rounds_id, event_type, event_data)
		VALUES ($1, $2, $3, $4)
	`, roomID, roundID, eventType, eventData)
	if err != nil {
		return fmt.Errorf("create room event: %w", err)
	}

	return nil
}

func (r *roomRepo) ListRecentRoomEvents(ctx context.Context, roomID int64, limit int) ([]domain.RoomEvent, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := r.db.Query(ctx, `
		SELECT room_event_id, room_id, rounds_id, event_type, event_data, created_at
		FROM room_events
		WHERE room_id = $1
		ORDER BY created_at DESC, room_event_id DESC
		LIMIT $2
	`, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent room events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.RoomEvent, 0, limit)
	for rows.Next() {
		var event domain.RoomEvent
		if err := rows.Scan(&event.RoomEventID, &event.RoomID, &event.RoundID, &event.EventType, &event.EventData, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan room event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}
