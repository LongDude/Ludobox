package presenters

import "time"

type GameServerResponse struct {
	ServerID        int        `json:"server_id"`
	InstanceKey     string     `json:"instance_key"`
	RedisHost       string     `json:"redis_host"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	LastHeartbeatAt time.Time  `json:"last_heartbeat_at"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
}

type GameServersResponse struct {
	Items    []GameServerResponse `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}
