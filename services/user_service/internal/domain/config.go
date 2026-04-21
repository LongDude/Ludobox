package domain

import "time"

type Config struct {
	ID                  int        `json:"config_id"`
	GameID              int        `json:"game_id"`
	Game                *Game      `json:"game,omitempty"`
	Capacity            int        `json:"capacity"`
	RegistrationPrice   int        `json:"registration_price"`
	IsBoost             bool       `json:"is_boost"`
	BoostPrice          int        `json:"boost_price"`
	BoostPower          int        `json:"boost_power"`
	NumberWinners       int        `json:"number_winners"`
	WinningDistribution []int      `json:"winning_distribution"`
	Commission          int        `json:"commission"`
	Time                int        `json:"time"`
	RoundTime           int        `json:"round_time"`
	NextRoundDelay      int        `json:"next_round_delay"`
	MinUsers            int        `json:"min_users"`
	ArchivedAt          *time.Time `json:"archived_at,omitempty"`
}
