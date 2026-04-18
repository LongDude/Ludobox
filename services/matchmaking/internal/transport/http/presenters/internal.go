package presenters

import "time"

type ServerResponse struct {
	Server_id      int64     `json:"server_id"`
	Instance_key   string    `json:"instance_key"`
	Redis_host     string    `json:"redis_host"`
	Active_rooms   int64     `json:"active_rooms"`
	Last_heartbeat time.Time `json:"last_heartbeat"`
}

type ResolveServerResponse struct {
	Server_id    int64  `json:"server_id"`
	Instance_key string `json:"instance_key"`
	Redis_host   string `json:"redis_host"`
}
