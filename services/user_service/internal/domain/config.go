package domain

import "time"

type Config struct {
	ID                   int        `json:"config_id"`
	Game_ID              int        `json:"game_id"`
	Capacity             int        `json:"capacity"`
	Registraton_price    int        `json:"registration_price"`
	Is_boost             bool       `json:"is_boost"`
	Boost_price          int        `json:"boost_price"`
	Boost_power          int        `json:"boost_power"`
	Number_winners       int        `json:"number_winners"`
	Winning_distribution []int      `json:"winning_distribution"`
	Commission           int        `json:"commission"`
	Time                 int        `json:"time"`
	Min_users            int        `json:"min_users"`
	Archived_at          *time.Time `json:"Archived_at"`
}
