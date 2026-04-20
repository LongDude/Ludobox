package repository

import (
	"context"
	"errors"
	"game_server/internal/domain"
)

var (
	ErrorUserNotFound     = errors.New("user not found")
	ErrorUserAlreadyExist = errors.New("user already exist")
)

type GameServerRegistrationParams struct {
	InstanceKey string
	RedisHost   string
}

type InternalRepository interface {
	RegisterGameServer(ctx context.Context, params GameServerRegistrationParams) (*domain.GameServer, error)
	HeartbeatGameServer(ctx context.Context, serverID int64) error
	DeactivateGameServer(ctx context.Context, serverID int64) error
}
type SessionRepository interface {
}
