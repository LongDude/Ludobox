package domain

import "time"

type UserRatingHistoryParams struct {
	DateFrom *time.Time
	DateTo   *time.Time
}

type UserRatingHistoryPoint struct {
	HistoryID   int64     `json:"history_id"`
	RoundID     *int64    `json:"round_id,omitempty"`
	RoomID      *int64    `json:"room_id,omitempty"`
	GameID      *int64    `json:"game_id,omitempty"`
	GameName    *string   `json:"game_name,omitempty"`
	Source      string    `json:"source"`
	Delta       int64     `json:"delta"`
	RatingAfter int64     `json:"rating_after"`
	Rank        string    `json:"rank"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserRatingHistory struct {
	CurrentRating int64                    `json:"current_rating"`
	CurrentRank   string                   `json:"current_rank"`
	PeriodChange  int64                    `json:"period_change"`
	Items         []UserRatingHistoryPoint `json:"items"`
}

func RankFromRating(rating int64) string {
	switch {
	case rating >= 5000:
		return "diamond"
	case rating >= 3000:
		return "platinum"
	case rating >= 1500:
		return "gold"
	case rating >= 500:
		return "silver"
	default:
		return "bronze"
	}
}
