package postgres

import (
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) repository.UserRepository {
	return &userRepository{db: db}
}

type configRepository struct {
	db *pgxpool.Pool
}

func NewConfigRepository(db *pgxpool.Pool) repository.ConfigRepository {
	return &configRepository{db: db}
}

type roomRepository struct {
	db *pgxpool.Pool
}

func NewRoomRepository(db *pgxpool.Pool) repository.RoomRepository {
	return &roomRepository{db: db}
}
