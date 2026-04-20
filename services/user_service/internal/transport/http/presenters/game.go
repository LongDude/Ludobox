package presenters

import "time"

type GameResponse struct {
	GameID     int        `json:"game_id"`
	NameGame   string     `json:"name_game"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

type GamesResponse struct {
	Items    []GameResponse `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type GameUpsertRequest struct {
	NameGame string `json:"name_game"`
}
