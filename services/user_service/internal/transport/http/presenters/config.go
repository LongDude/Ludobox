package presenters

import "time"

type ConfigResponse struct {
	ConfigID            int           `json:"config_id"`
	GameID              int           `json:"game_id"`
	Game                *GameResponse `json:"game,omitempty"`
	Capacity            int           `json:"capacity"`
	RegistrationPrice   int           `json:"registration_price"`
	IsBoost             bool          `json:"is_boost"`
	BoostPrice          int           `json:"boost_price"`
	BoostPower          int           `json:"boost_power"`
	NumberWinners       int           `json:"number_winners"`
	WinningDistribution []int         `json:"winning_distribution"`
	Commission          int           `json:"commission"`
	Time                int           `json:"time"`
	MinUsers            int           `json:"min_users"`
	ArchivedAt          *time.Time    `json:"archived_at,omitempty"`
}

type ConfigsResponse struct {
	Items    []ConfigResponse `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type ConfigUpsertRequest struct {
	GameID              int   `json:"game_id"`
	Capacity            int   `json:"capacity"`
	RegistrationPrice   int   `json:"registration_price"`
	IsBoost             bool  `json:"is_boost"`
	BoostPrice          int   `json:"boost_price"`
	BoostPower          int   `json:"boost_power"`
	NumberWinners       int   `json:"number_winners"`
	WinningDistribution []int `json:"winning_distribution"`
	Commission          int   `json:"commission"`
	Time                int   `json:"time"`
	MinUsers            int   `json:"min_users"`
}
