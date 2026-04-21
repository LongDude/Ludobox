package dto

import "time"

// JoinRoomRequest - запрос на присоединение к комнате
type JoinRoomRequest struct {
	RoomID int64 `json:"room_id" binding:"required"`
}

// JoinRoomWithSeatRequest - запрос на присоединение к комнате с выбором места
type JoinRoomWithSeatRequest struct {
	RoomID       int64 `json:"room_id" binding:"required"`
	NumberInRoom int   `json:"number_in_room" binding:"required,min=1"`
}

// JoinRoomResponse - ответ при успешном присоединении
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

// PurchaseBoostRequest - запрос на покупку буста
type PurchaseBoostRequest struct {
	BoostValue int64 `json:"boost_value" binding:"required,min=1"`
}

// PurchaseBoostResponse - ответ при успешной покупке буста
type PurchaseBoostResponse struct {
	Success    bool   `json:"success"`
	BoostPower int    `json:"boost_power"`
	BoostCost  int64  `json:"boost_cost"`
	Message    string `json:"message,omitempty"`
}

// CancelBoostResponse - ответ при отмене буста
type CancelBoostResponse struct {
	Success bool   `json:"success"`
	Refund  int64  `json:"refund"`
	Message string `json:"message,omitempty"`
}

// LeaveRoomRequest - запрос на выход из комнаты
type LeaveRoomRequest struct {
	RoundID int64 `json:"round_id" binding:"required"`
}

// LeaveRoomResponse - ответ при выходе из комнаты
type LeaveRoomResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Refund  int64  `json:"refund"`
}

// RoundStatusResponse - информация о статусе раунда
type RoundStatusResponse struct {
	RoundID         int64             `json:"round_id"`
	Status          string            `json:"status"`
	Participants    []ParticipantInfo `json:"participants"`
	TimeLeftSeconds int               `json:"time_left_seconds"`
	CreatedAt       time.Time         `json:"created_at"`
	Winners         []ParticipantInfo `json:"winners,omitempty"`
}

// ParticipantInfo - информация об участнике
type ParticipantInfo struct {
	ParticipantID int64      `json:"participant_id"`
	UserID        *int64     `json:"user_id,omitempty"` // nil для ботов
	NumberInRoom  int        `json:"number_in_room"`
	Boost         int        `json:"boost"`
	WinningMoney  int64      `json:"winning_money,omitempty"`
	IsBot         bool       `json:"is_bot"`
	ExitedAt      *time.Time `json:"exited_at,omitempty"`
}

// SSEEvent - событие для SSE
type SSEEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// EventPlayerJoined - событие присоединения игрока
type EventPlayerJoined struct {
	ParticipantID  int64 `json:"participant_id"`
	NumberInRoom   int   `json:"number_in_room"`
	CurrentPlayers int   `json:"current_players"`
}

// EventPlayerLeft - событие выхода игрока
type EventPlayerLeft struct {
	ParticipantID  int64 `json:"participant_id"`
	NumberInRoom   int   `json:"number_in_room"`
	CurrentPlayers int   `json:"current_players"`
}

// EventBoostPurchased - событие покупки буста
type EventBoostPurchased struct {
	ParticipantID int64 `json:"participant_id"`
	BoostPower    int   `json:"boost_power"`
}

// EventBoostCancelled - событие отмены буста
type EventBoostCancelled struct {
	ParticipantID int64 `json:"participant_id"`
}

// EventRoundStarted - событие начала игры
type EventRoundStarted struct {
	RoundID         int64 `json:"round_id"`
	FinalPlayers    int   `json:"final_players"`
	GameDurationSec int   `json:"game_duration_sec"`
}

// EventRoundFinalized - событие завершения игры
type EventRoundFinalized struct {
	RoundID int64           `json:"round_id"`
	Winners []WinnerInfo    `json:"winners"`
	Payouts map[int64]int64 `json:"payouts"` // participantID -> winnings
}

// WinnerInfo - информация о победителе
type WinnerInfo struct {
	ParticipantID int64 `json:"participant_id"`
	NumberInRoom  int   `json:"number_in_room"`
	Winnings      int64 `json:"winnings"`
}

// ErrorResponse - стандартный ответ об ошибке
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
