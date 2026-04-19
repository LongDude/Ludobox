package domain

import "time"

type Room struct {
	ID           int        `json:"room_id"`
	ConfigID     int        `json:"config_id"`
	GameServerID int        `json:"server_id"`
	Status       RoomStatus `json:"status"`
	ArchivedAt   time.Time  `json:"archived_at"`
}

type RoomStatus string

const (
	StatusOpen      RoomStatus = "open"
	StatusInGame    RoomStatus = "in_game"
	StatusCompleted RoomStatus = "completed"
)
