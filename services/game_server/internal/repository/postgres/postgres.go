package postgres

import (
	"game_server/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type internalRepository struct {
	db *pgxpool.Pool
}

func NewInternalRepository(db *pgxpool.Pool) repository.InternalRepository {
	return &internalRepository{db: db}
}
