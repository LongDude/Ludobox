package domain

import "time"

type Room struct {
	ID             int        `json:"room_id"`
	ConfigID       int        `json:"config_id"`
	Config         *Config    `json:"config,omitempty"`
	GameServerID   int        `json:"server_id"`
	ServerName     string     `json:"server_name,omitempty"`
	CurrentPlayers int        `json:"current_players"`
	Status         RoomStatus `json:"status"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`
}

type RoomStatus string

const (
	StatusOpen      RoomStatus = "open"
	StatusInGame    RoomStatus = "in_game"
	StatusCompleted RoomStatus = "completed"
)
