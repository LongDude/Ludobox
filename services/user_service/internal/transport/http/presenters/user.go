package presenters

type TestResponse struct {
	Pong string `json:"pong" binding:"required" validate:"required"`
}

type UserResponse struct {
	UserID   int    `json:"user_id" example:"42"`
	Nickname string `json:"nickname" example:"user_42"`
	Balance  int    `json:"balance" example:"1000"`
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
