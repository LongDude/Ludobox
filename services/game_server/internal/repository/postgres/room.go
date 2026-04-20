package postgres

import (
	"context"
	"fmt"

	"game_server/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type roomRepo struct {
	db *pgxpool.Pool
}

func NewRoomRepository(db *pgxpool.Pool) repository.RoomRepository {
	return &roomRepo{db: db}
}

func (r *roomRepo) InTransaction(ctx context.Context, fn func(ts repository.TransactionScope) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ts := &txScope{tx: tx}
	if err := fn(ts); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type txScope struct {
	tx pgx.Tx
}
