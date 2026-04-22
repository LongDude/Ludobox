package presenters

import "time"

type UserResponse struct {
	UserID   int    `json:"user_id" example:"42"`
	Nickname string `json:"nickname" example:"user_42"`
	Balance  int    `json:"balance" example:"1000"`
	Rating   int64  `json:"rating" example:"320"`
	Rank     string `json:"rank" example:"bronze"`
}

type UserCreateRequest struct {
	Nickname *string `json:"nickname,omitempty" example:"ludobox_vip"`
	Balance  *int    `json:"balance,omitempty" example:"5000"`
}

type UserUpdateRequest struct {
	Nickname *string `json:"nickname" example:"ludobox_vip_2"`
}

type UserBalanceUpdateRequest struct {
	Delta int `json:"delta" example:"-250"`
}

type UserRatingHistoryPointResponse struct {
	HistoryID   int64     `json:"history_id"`
	RoundID     *int64    `json:"round_id,omitempty"`
	RoomID      *int64    `json:"room_id,omitempty"`
	GameID      *int64    `json:"game_id,omitempty"`
	GameName    *string   `json:"game_name,omitempty"`
	Source      string    `json:"source"`
	Delta       int64     `json:"delta"`
	RatingAfter int64     `json:"rating_after"`
	Rank        string    `json:"rank"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserRatingHistoryResponse struct {
	CurrentRating int64                            `json:"current_rating"`
	CurrentRank   string                           `json:"current_rank"`
	PeriodChange  int64                            `json:"period_change"`
	Items         []UserRatingHistoryPointResponse `json:"items"`
}
