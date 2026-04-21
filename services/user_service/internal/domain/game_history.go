package domain

import "time"

type GameHistoryParams struct {
	Pagination
	GameID   *int
	RoomID   *int
	Status   string
	DateFrom *time.Time
	DateTo   *time.Time
}

type GameHistoryItem struct {
	RoundID       int64      `json:"round_id"`
	ParticipantID int64      `json:"participant_id"`
	RoomID        int64      `json:"room_id"`
	GameID        int64      `json:"game_id"`
	GameName      string     `json:"game_name"`
	SeatNumber    int        `json:"seat_number"`
	RoundStatus   string     `json:"round_status"`
	Result        string     `json:"result"`
	EntryFee      int64      `json:"entry_fee"`
	BoostFee      int64      `json:"boost_fee"`
	WinningMoney  int64      `json:"winning_money"`
	NetResult     int64      `json:"net_result"`
	JoinedAt      time.Time  `json:"joined_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}
