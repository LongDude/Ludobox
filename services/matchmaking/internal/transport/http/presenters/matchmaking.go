package presenters

type RoomRecommendationResponse struct {
	RoomID            int64   `json:"room_id"`
	ConfigID          int64   `json:"config_id"`
	ServerID          int64   `json:"server_id"`
	GameID            int64   `json:"game_id"`
	RegistrationPrice int64   `json:"registration_price"`
	Capacity          int32   `json:"capacity"`
	MinUsers          int32   `json:"min_users"`
	IsBoost           bool    `json:"is_boost"`
	BoostPower        int32   `json:"boost_power"`
	CurrentPlayers    int32   `json:"current_players"`
	InstanceKey       string  `json:"instance_key"`
	RedisHost         string  `json:"redis_host"`
	Score             float64 `json:"score"`
}

type RecommendRoomsResponse struct {
	Items  []RoomRecommendationResponse `json:"items"`
	Cached bool                         `json:"cached"`
}

type QuickMatchResponse struct {
	Room               RoomRecommendationResponse `json:"room"`
	RoundID            int64                      `json:"round_id"`
	RoundParticipantID int64                      `json:"round_participant_id"`
	SeatNumber         int32                      `json:"seat_number"`
	ReusedExistingRoom bool                       `json:"reused_existing_room"`
}
