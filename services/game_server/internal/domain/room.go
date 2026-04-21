package domain

import "time"

type RoomStatus string

const (
	RoomStatusOpen      RoomStatus = "open"
	RoomStatusInGame    RoomStatus = "in_game"
	RoomStatusCompleted RoomStatus = "completed"
)

type Room struct {
	RoomID         int64      `json:"room_id"`
	ConfigID       int64      `json:"config_id"`
	ServerID       int64      `json:"server_id"`
	Status         RoomStatus `json:"status"`
	CurrentPlayers int        `json:"current_players"`
	ArchivedAt     *time.Time `json:"archived_at"`
}

type RoomConfig struct {
	ConfigID            int64      `json:"config_id"`
	GameID              int64      `json:"game_id"`
	Capacity            int        `json:"capacity"`
	RegistrationPrice   int64      `json:"registration_price"`
	IsBoost             bool       `json:"is_boost"`
	BoostPrice          int64      `json:"boost_price"`
	BoostPower          int        `json:"boost_power"`
	NumberWinners       int        `json:"number_winners"`
	WinningDistribution []int      `json:"winning_distribution"`
	Commission          int        `json:"commission"`
	Time                int        `json:"time"` // In seconds
	MinUsers            int        `json:"min_users"`
	ArchivedAt          *time.Time `json:"archived_at"`
}

type RoomInfo struct {
	Room                    *Room       `json:"room"`
	Config                  *RoomConfig `json:"config"`
	CurrentRoundID          *int64      `json:"current_round_id,omitempty"`
	CurrentRoundStatus      *string     `json:"current_round_status,omitempty"`
	ActiveParticipantsCount int         `json:"active_participants_count"`
}
