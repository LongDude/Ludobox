package postgres

import (
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type adminRepository struct {
	db *pgxpool.Pool
}

func NewAdminRepository(db *pgxpool.Pool) repository.AdminRepository {
	return &adminRepository{db: db}
}
