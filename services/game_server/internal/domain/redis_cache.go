package domain

import "time"

// RoundStatus represents the state of a round
type RoundStatusEnum string

const (
	RoundStatusWaiting   RoundStatusEnum = "waiting"
	RoundStatusActive    RoundStatusEnum = "active"
	RoundStatusFinished  RoundStatusEnum = "finished"
	RoundStatusCancelled RoundStatusEnum = "cancelled"
)

// RoomCacheData stores room information in Redis
type RoomCacheData struct {
	RoomID        int64             `json:"room_id"`
	ConfigID      int64             `json:"config_id"`
	ServerID      int64             `json:"server_id"`
	CurrentRoundID *int64            `json:"current_round_id,omitempty"`
	Status        RoomStatus        `json:"status"`
}

// RoundCacheData stores round information in Redis
type RoundCacheData struct {
	RoundID       int64             `json:"round_id"`
	RoomID        int64             `json:"room_id"`
	Status        RoundStatusEnum   `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	Participants  map[int64]RoundParticipantCacheData `json:"participants"` // map[participantID]ParticipantData
}

// RoundParticipantCacheData stores participant data in round cache
type RoundParticipantCacheData struct {
	RoundParticipantID int64      `json:"round_participant_id"`
	UserID             int64      `json:"user_id"`
	NumberInRoom       int        `json:"number_in_room"`
	Boost              int        `json:"boost"`
	ExitRoomAt         *time.Time `json:"exit_room_at,omitempty"`
}

// UserSessionData stores user session information in Redis
type UserSessionData struct {
	UserID         int64  `json:"user_id"`
	RoomID         int64  `json:"room_id"`
	RoundID        int64  `json:"round_id"`
	ConfigID       int64  `json:"config_id"`
	ParticipantID  int64  `json:"participant_id"`
	ConnectedAt    time.Time `json:"connected_at"`
}

// RoomConfigCacheData stores room configuration in Redis
type RoomConfigCacheData struct {
	ConfigID            int64   `json:"config_id"`
	GameID              int64   `json:"game_id"`
	Capacity            int     `json:"capacity"`
	RegistrationPrice   int64   `json:"registration_price"`
	IsBoost             bool    `json:"is_boost"`
	BoostPrice          int64   `json:"boost_price"`
	BoostPower          int     `json:"boost_power"`
	NumberWinners       int     `json:"number_winners"`
	WinningDistribution []int   `json:"winning_distribution"`
	Commission          int     `json:"commission"`
	Time                int     `json:"time"` // In seconds
	MinUsers            int     `json:"min_users"`
}
