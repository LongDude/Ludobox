package service

import (
	"context"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type RoomService interface {
	CreateRoomByConfigID(ctx context.Context, config_id int) (*domain.Room, error)
	GetRoomByID(ctx context.Context, id int) (*domain.Room, error)
	GetNotArchivedRooms(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Room], error)
	UpdateRoomByID(ctx context.Context, id int, room *domain.Room) (*domain.Room, error)
	DeleteRoomByID(ctx context.Context, id int) error
}

type roomService struct {
	configRepository repository.ConfigRepository
	logger           *logrus.Logger
}
