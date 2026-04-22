package presenters

import "time"

type GameHistoryItemResponse struct {
	RoundID            int64      `json:"round_id"`
	RoomID             int64      `json:"room_id"`
	GameID             int64      `json:"game_id"`
	GameName           string     `json:"game_name"`
	RoundStatus        string     `json:"round_status"`
	Result             string     `json:"result"`
	ReservedSeats      []int      `json:"reserved_seats"`
	WinningSeats       []int      `json:"winning_seats"`
	ReservedSeatsCount int        `json:"reserved_seats_count"`
	WinningSeatsCount  int        `json:"winning_seats_count"`
	EntryFee           int64      `json:"entry_fee"`
	BoostFee           int64      `json:"boost_fee"`
	TotalSpent         int64      `json:"total_spent"`
	WinningMoney       int64      `json:"winning_money"`
	NetResult          int64      `json:"net_result"`
	JoinedAt           time.Time  `json:"joined_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
}

type GameHistoryResponse struct {
	Items    []GameHistoryItemResponse `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}
