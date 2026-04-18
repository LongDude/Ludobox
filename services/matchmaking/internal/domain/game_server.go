package domain

import (
	"errors"
	"time"
)

var (
	ErrorRoomNotFound          = errors.New("room not found")
	ErrorNoActiveGameServers   = errors.New("no active game servers")
	ErrorGameServerUnavailable = errors.New("game server unavailable")
)

type GameServer struct {
	ServerID        int64
	InstanceKey     string
	RedisHost       string
	Status          string
	LastHeartbeatAt time.Time
	ArchivedAt      *time.Time
	ActiveRooms     int64
}
