package domain

import "time"

type Game struct {
	ID         int        `json:"game_id"`
	Name       string     `json:"name_game"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}
