package dto

type RecommendRoomsRequest struct {
	GameID               *int64 `json:"game_id,omitempty"`
	MinRegistrationPrice *int64 `json:"min_registration_price,omitempty"`
	MaxRegistrationPrice *int64 `json:"max_registration_price,omitempty"`
	MinCapacity          *int32 `json:"min_capacity,omitempty"`
	MaxCapacity          *int32 `json:"max_capacity,omitempty"`
	IsBoost              *bool  `json:"is_boost,omitempty"`
	MinBoostPower        *int32 `json:"min_boost_power,omitempty"`
	Limit                int32  `json:"limit,omitempty"`
}

type QuickMatchRequest struct {
	GameID               *int64 `json:"game_id,omitempty"`
	MinRegistrationPrice *int64 `json:"min_registration_price,omitempty"`
	MaxRegistrationPrice *int64 `json:"max_registration_price,omitempty"`
	MinCapacity          *int32 `json:"min_capacity,omitempty"`
	MaxCapacity          *int32 `json:"max_capacity,omitempty"`
	IsBoost              *bool  `json:"is_boost,omitempty"`
	MinBoostPower        *int32 `json:"min_boost_power,omitempty"`
}
