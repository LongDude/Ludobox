package domain

import (
	"encoding/json"
	"time"
)

type RoomEvent struct {
	RoomEventID int64           `json:"room_event_id"`
	RoomID      int64           `json:"room_id"`
	RoundID     *int64          `json:"round_id,omitempty"`
	EventType   string          `json:"event_type"`
	EventData   json.RawMessage `json:"event_data"`
	CreatedAt   time.Time       `json:"created_at"`
}
