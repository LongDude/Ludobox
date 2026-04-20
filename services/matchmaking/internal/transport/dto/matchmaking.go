package dto

type RecommendRoomsRequest struct {
	GameID               *int64 `json:"game_id,omitempty" form:"game_id"`
	MinRegistrationPrice *int64 `json:"min_registration_price,omitempty" form:"min_registration_price"`
	MaxRegistrationPrice *int64 `json:"max_registration_price,omitempty" form:"max_registration_price"`
	MinCapacity          *int32 `json:"min_capacity,omitempty" form:"min_capacity"`
	MaxCapacity          *int32 `json:"max_capacity,omitempty" form:"max_capacity"`
	IsBoost              *bool  `json:"is_boost,omitempty" form:"is_boost"`
	MinBoostPower        *int32 `json:"min_boost_power,omitempty" form:"min_boost_power"`
	Page                 int32  `json:"page,omitempty" form:"page"`
	PageSize             int32  `json:"page_size,omitempty" form:"page_size"`
}

type QuickMatchRequest struct {
	GameID               *int64 `json:"game_id,omitempty" form:"game_id"`
	MinRegistrationPrice *int64 `json:"min_registration_price,omitempty" form:"min_registration_price"`
	MaxRegistrationPrice *int64 `json:"max_registration_price,omitempty" form:"max_registration_price"`
	MinCapacity          *int32 `json:"min_capacity,omitempty" form:"min_capacity"`
	MaxCapacity          *int32 `json:"max_capacity,omitempty" form:"max_capacity"`
	IsBoost              *bool  `json:"is_boost,omitempty" form:"is_boost"`
	MinBoostPower        *int32 `json:"min_boost_power,omitempty" form:"min_boost_power"`
}
