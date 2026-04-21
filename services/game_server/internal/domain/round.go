package domain

import "time"

type Round struct {
	RoundsID   int64      `json:"rounds_id"`
	RoomID     int64      `json:"room_id"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ArchivedAt *time.Time `json:"archived_at"`
}

type RoundParticipant struct {
	RoundParticipantID int64      `json:"round_participants_id"`
	UserID             int64      `json:"user_id"`
	RoundsID           int64      `json:"rounds_id"`
	Boost              int        `json:"boost"`
	WinningMoney       int64      `json:"winning_money"`
	NumberInRoom       int        `json:"number_in_room"`
	IsBot              bool       `json:"is_bot"`
	ExitRoomAt         *time.Time `json:"exit_room_at"`
}
