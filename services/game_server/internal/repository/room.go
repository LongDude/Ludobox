package repository

import (
	"context"
	"time"

	"game_server/internal/domain"
)

// TransactionScope предоставляет методы для работы внутри одной транзакции.
// Сервис видит только этот интерфейс, никаких pgx/pgxpool.
type TransactionScope interface {
	// Баланс
	GetBalanceLocked(ctx context.Context, userID int64) (int64, error)
	UpdateBalance(ctx context.Context, userID int64, delta int64) error

	// Участники
	CreateParticipant(ctx context.Context, userID, roundID int64, numberInRoom int) (int64, error)
	GetParticipantUserID(ctx context.Context, participantID int64) (int64, error)
	UpdateParticipantBoost(ctx context.Context, participantID int64, boost int) error
	MarkParticipantExited(ctx context.Context, participantID int64) error
	UpdateWinningMoney(ctx context.Context, participantID int64, amount int64) error

	// Резервы
	ReserveEntry(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error)
	ReserveBoost(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error)
	ReleaseAllReservations(ctx context.Context, participantID int64) (int64, error)
	ReleaseBoostReservations(ctx context.Context, participantID int64) (int64, error)
	CommitReservations(ctx context.Context, participantID int64) (int64, error)

	// Раунды
	ArchiveRound(ctx context.Context, roundID int64) error
}

// RoomRepository управляет транзакциями и предоставляет read-only доступ к данным.
type RoomRepository interface {
	// InTransaction открывает транзакцию и передаёт в коллбэк безопасный TransactionScope.
	// Если коллбэк возвращает ошибку, происходит автоматический Rollback.
	InTransaction(ctx context.Context, fn func(ts TransactionScope) error) error

	// Read-only методы (не требуют транзакции, если не указано иное)
	GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error)
	GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error)
}
