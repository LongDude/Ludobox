package repository

import (
	"context"
	"errors"
	"user_service/internal/domain"
)

var (
	ErrorUserNotFound        = errors.New("user not found")
	ErrorUserAlreadyExist    = errors.New("user already exist")
	ErrorNegativeBalance     = errors.New("balance cannot be negative")
	ErrorConfigNotFound      = errors.New("config not found")
	ErrorConfigArchived      = errors.New("config is archived")
	ErrorInvalidConfig       = errors.New("config is invalid")
	ErrorInvalidListParams   = errors.New("invalid list params")
	ErrorRoomNotFound        = errors.New("room not found")
	ErrorRoomArchived        = errors.New("room is archived")
	ErrorInvalidRoom         = errors.New("room is invalid")
	ErrorNoActiveGameServers = errors.New("no active game servers")
)

type UserRepository interface {
	GetUserByID(ctx context.Context, id int) (*domain.User, error)
	CreateUserByID(ctx context.Context, id int) (*domain.User, error)
	UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error)
	DeleteUserByID(ctx context.Context, id int) error
}
type ConfigRepository interface {
	GetConfigs(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Config], error)
	GetConfigByID(ctx context.Context, id int) (*domain.Config, error)
	CreateNewConfig(ctx context.Context, config *domain.Config) (*domain.Config, error)
	UpdateConfigByID(ctx context.Context, id int, config *domain.Config) (*domain.Config, error)
	DeleteConfigByID(ctx context.Context, id int) error
}
type RoomRepository interface {
	CreateRoomByConfigID(ctx context.Context, configID int, serverID int) (*domain.Room, error)
	GetRoomByID(ctx context.Context, id int) (*domain.Room, error)
	GetNotArchivedRooms(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Room], error)
	UpdateRoomByID(ctx context.Context, id int, room *domain.Room) (*domain.Room, error)
	DeleteRoomByID(ctx context.Context, id int) error
}
type GameServerRepository interface {
	GetServers(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.GameServer], error)
}
type SessionRepository interface {
}
