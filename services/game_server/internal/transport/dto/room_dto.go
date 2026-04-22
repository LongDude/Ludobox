package dto

import "time"

type JoinRoomWithSeatRequest struct {
	NumberInRoom int `json:"number_in_room" binding:"required,min=1"`
}

type JoinRoomResponse struct {
	ParticipantID  int64      `json:"participant_id"`
	RoundID        int64      `json:"round_id"`
	NumberInRoom   int        `json:"number_in_room"`
	RoomCapacity   int        `json:"room_capacity"`
	CurrentPlayers int        `json:"current_players"`
	MinPlayers     int        `json:"min_players"`
	RoundStatus    string     `json:"round_status"`
	EntryPrice     int64      `json:"entry_price"`
	TimerStartsAt  *time.Time `json:"timer_starts_at,omitempty"`
}

type RoomStateResponse struct {
	RoomID                  int64             `json:"room_id"`
	RoundID                 int64             `json:"round_id"`
	RoomCapacity            int               `json:"room_capacity"`
	CurrentPlayers          int               `json:"current_players"`
	MinPlayers              int               `json:"min_players"`
	EntryPrice              int64             `json:"entry_price"`
	RoundStatus             string            `json:"round_status"`
	IsBoost                 bool              `json:"is_boost"`
	BoostPower              int               `json:"boost_power"`
	BoostPrice              int64             `json:"boost_price"`
	WaitingTime             int               `json:"waiting_time"`
	RoundTime               int               `json:"round_time"`
	NextRoundDelay          int               `json:"next_round_delay"`
	TimerStartsAt           *time.Time        `json:"timer_starts_at,omitempty"`
	CurrentUserParticipants []ParticipantInfo `json:"current_user_participants,omitempty"`
	RecentEvents            []SSEEvent        `json:"recent_events,omitempty"`
}

type PurchaseBoostResponse struct {
	Success    bool   `json:"success"`
	BoostPower int    `json:"boost_power"`
	BoostCost  int64  `json:"boost_cost"`
	Message    string `json:"message,omitempty"`
}

type CancelBoostResponse struct {
	Success bool   `json:"success"`
	Refund  int64  `json:"refund"`
	Message string `json:"message,omitempty"`
}

type LeaveRoomRequest struct {
	RoundID int64 `json:"round_id" binding:"required"`
}

type LeaveRoomResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Refund  int64  `json:"refund"`
}

type RoundStatusResponse struct {
	RoundID         int64             `json:"round_id"`
	Status          string            `json:"status"`
	Participants    []ParticipantInfo `json:"participants"`
	TimeLeftSeconds int               `json:"time_left_seconds"`
	CreatedAt       time.Time         `json:"created_at"`
	Winners         []ParticipantInfo `json:"winners,omitempty"`
}

type ParticipantInfo struct {
	ParticipantID int64      `json:"participant_id"`
	UserID        *int64     `json:"user_id,omitempty"`
	Nickname      *string    `json:"nickname,omitempty"`
	NumberInRoom  int        `json:"number_in_room"`
	Boost         int        `json:"boost"`
	WinningMoney  int64      `json:"winning_money,omitempty"`
	IsBot         bool       `json:"is_bot"`
	ExitedAt      *time.Time `json:"exited_at,omitempty"`
}

type SSEEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type EventPlayerJoined struct {
	ParticipantID  int64 `json:"participant_id"`
	NumberInRoom   int   `json:"number_in_room"`
	CurrentPlayers int   `json:"current_players"`
}

type EventPlayerLeft struct {
	ParticipantID  int64 `json:"participant_id"`
	NumberInRoom   int   `json:"number_in_room"`
	CurrentPlayers int   `json:"current_players"`
}

type EventBoostPurchased struct {
	ParticipantID int64 `json:"participant_id"`
	BoostPower    int   `json:"boost_power"`
}

type EventBoostCancelled struct {
	ParticipantID int64 `json:"participant_id"`
}

type EventRoundTimer struct {
	RoundID     int64  `json:"round_id"`
	Status      string `json:"status"`
	SecondsLeft int    `json:"seconds_left"`
}

type EventRoundStarted struct {
	RoundID         int64 `json:"round_id"`
	FinalPlayers    int   `json:"final_players"`
	GameDurationSec int   `json:"game_duration_sec"`
}

type EventRoundFinalized struct {
	RoundID        int64           `json:"round_id"`
	Winners        []WinnerInfo    `json:"winners"`
	Payouts        map[int64]int64 `json:"payouts"`
	NextRoundID    int64           `json:"next_round_id,omitempty"`
	NextRoundDelay int             `json:"next_round_delay,omitempty"`
}

type WinnerInfo struct {
	ParticipantID int64   `json:"participant_id"`
	UserID        *int64  `json:"user_id,omitempty"`
	Nickname      *string `json:"nickname,omitempty"`
	NumberInRoom  int     `json:"number_in_room"`
	Winnings      int64   `json:"winnings"`
	GrossWinnings int64   `json:"gross_winnings"`
	IsBot         bool    `json:"is_bot"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
