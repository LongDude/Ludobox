package config

import "time"

const (
	// ReservationGracePeriod is the grace period for reservation expiration (60 seconds)
	ReservationGracePeriod = 60 * time.Second

	// RoundWaitingTimeout is the maximum time to wait for min_users before cancelling the round (15 minutes)
	RoundWaitingTimeout = 15 * time.Minute

	// RedisSessionTTL is the TTL for user session data in Redis (1 hour)
	RedisSessionTTL = 1 * time.Hour

	// RedisRoomCacheTTL is the TTL for room cache data in Redis (1 hour)
	RedisRoomCacheTTL = 1 * time.Hour
)

// Redis key prefixes
const (
	RedisKeyGameServerRooms  = "game_server:%d:rooms"
	RedisKeyRoom             = "room:%d"
	RedisKeyRoomConfig       = "room_config:%d"
	RedisKeyRound            = "round:%d"
	RedisKeyRoundParticipants = "round:%d:participants"
	RedisKeyUserSession      = "user_session:%d"
)
