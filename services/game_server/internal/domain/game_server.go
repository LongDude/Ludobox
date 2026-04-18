package domain

import "time"

type GameServer struct {
	ServerID        int64
	InstanceKey     string
	RedisHost       string
	Status          string
	LastHeartbeatAt time.Time
	ArchivedAt      *time.Time
}
