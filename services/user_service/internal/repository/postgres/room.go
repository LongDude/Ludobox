package postgres

import (
	"context"
	"user_service/internal/domain"
)

// CreateRoomByConfigID implements [repository.RoomRepository].
func (r *roomRepository) CreateRoomByConfigID(ctx context.Context, config_id int) (*domain.Room, error) {
	panic("unimplemented")
}

// DeleteRoomByID implements [repository.RoomRepository].
func (r *roomRepository) DeleteRoomByID(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetNotArchivedRooms implements [repository.RoomRepository].
func (r *roomRepository) GetNotArchivedRooms(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Room], error) {
	panic("unimplemented")
}

// GetRoomByID implements [repository.RoomRepository].
func (r *roomRepository) GetRoomByID(ctx context.Context, id int) (*domain.Room, error) {
	panic("unimplemented")
}

// UpdateRoomByID implements [repository.RoomRepository].
func (r *roomRepository) UpdateRoomByID(ctx context.Context, id int, room *domain.Room) (*domain.Room, error) {
	panic("unimplemented")
}
