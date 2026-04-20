package presenters

import "time"

type RoomResponse struct {
	RoomID     int             `json:"room_id"`
	ConfigID   int             `json:"config_id"`
	Config     *ConfigResponse `json:"config,omitempty"`
	ServerID   int             `json:"server_id"`
	ServerName string          `json:"server_name,omitempty"`
	Status     string          `json:"status"`
	ArchivedAt *time.Time      `json:"archived_at,omitempty"`
}

type RoomsResponse struct {
	Items    []RoomResponse `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type RoomCreateRequest struct {
	ConfigID int `json:"config_id"`
}

type RoomUpdateRequest struct {
	ServerID   *int       `json:"server_id,omitempty"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}
