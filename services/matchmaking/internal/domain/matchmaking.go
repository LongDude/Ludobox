package domain

import "time"

var (
	ErrorNoAvailableRooms         = ErrorWithMessage("no available rooms")
	ErrorActiveRoomNotFound       = ErrorWithMessage("active room not found")
	ErrorRoomUnavailable          = ErrorWithMessage("room is unavailable")
	ErrorRoomFull                 = ErrorWithMessage("room is full")
	ErrorUserAlreadyInRoom        = ErrorWithMessage("user is already in a room")
	ErrorInvalidMatchmakingParams = ErrorWithMessage("invalid matchmaking params")
	ErrorUserNotFound             = ErrorWithMessage("user not found")
)

type MatchmakingPreferences struct {
	UserID               int64
	GameID               *int64
	MinRegistrationPrice *int64
	MaxRegistrationPrice *int64
	MinCapacity          *int32
	MaxCapacity          *int32
	IsBoost              *bool
	MinBoostPower        *int32
	Limit                int32
	StaleAfter           time.Duration
}

type RoomRecommendation struct {
	RoomID            int64
	ConfigID          int64
	ServerID          int64
	GameID            int64
	RegistrationPrice int64
	Capacity          int32
	MinUsers          int32
	IsBoost           bool
	BoostPower        int32
	CurrentPlayers    int32
	InstanceKey       string
	RedisHost         string
	Score             float64
}

type RoomMembership struct {
	RoomRecommendation
	RoundID            int64
	RoundParticipantID int64
	SeatNumber         int32
}

type QuickMatchResult struct {
	RoomMembership
	ReusedExistingRoom bool
}

type ErrorWithMessage string

func (e ErrorWithMessage) Error() string {
	return string(e)
}
