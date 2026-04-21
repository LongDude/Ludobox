package postgres

import (
	"net/http"
	"time"
	"user_service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type internalRepository struct {
	db         *pgxpool.Pool
	httpClient *http.Client
}

func NewInternalRepository(db *pgxpool.Pool) repository.InternalRepository {
	return &internalRepository{
		db: db,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}
