package repository

import (
	"context"
	"encoding/json"
	"time"

	"game_server/internal/domain"
)

// TransactionScope provides data access methods inside a single transaction.
type TransactionScope interface {
	// Room / round state
	GetRoomForUpdate(ctx context.Context, roomID int64) (*domain.RoomInfo, error)
	GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error)
	GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error)
	GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error)
	CountUserActiveParticipants(ctx context.Context, roundID, userID int64) (int, error)
	IsSeatOccupied(ctx context.Context, roundID int64, numberInRoom int) (bool, error)

	// Balance
	GetBalanceLocked(ctx context.Context, userID int64) (int64, error)
	UpdateBalance(ctx context.Context, userID int64, delta int64) error

	// Participants
	CreateParticipant(ctx context.Context, userID, roundID int64, numberInRoom int) (int64, error)
	GetParticipantUserID(ctx context.Context, participantID int64) (int64, error)
	UpdateParticipantBoost(ctx context.Context, participantID int64, boost int) error
	MarkParticipantExited(ctx context.Context, participantID int64) error
	UpdateWinningMoney(ctx context.Context, participantID int64, amount int64) error
	ApplyUserRatingReward(ctx context.Context, reward domain.UserRatingReward) error

	// Reservations
	ReserveEntry(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error)
	ReserveBoost(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error)
	ReleaseAllReservations(ctx context.Context, participantID int64) (int64, error)
	ReleaseBoostReservations(ctx context.Context, participantID int64) (int64, error)
	CommitReservations(ctx context.Context, participantID int64) (int64, error)

	// Rounds
	ArchiveRound(ctx context.Context, roundID int64) error
	CreateRound(ctx context.Context, roomID int64) (int64, error)
	UpdateRoundStatus(ctx context.Context, roundID int64, status string) error
	GetActiveParticipantsCount(ctx context.Context, roundID int64) (int, error)
	FindFreeNumberInRoom(ctx context.Context, roundID int64, capacity int) (int, error)
	GetRoundStatus(ctx context.Context, roundID int64) (string, error)
}

// RoomRepository manages transactions and read-only access to room data.
type RoomRepository interface {
	InTransaction(ctx context.Context, fn func(ts TransactionScope) error) error

	GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error)
	GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error)
	GetActiveParticipantsByRoomAndUser(ctx context.Context, roomID, userID int64) ([]domain.RoundParticipant, error)
	CreateRoomEvent(ctx context.Context, roomID int64, roundID *int64, eventType string, eventData json.RawMessage) error
	ListRecentRoomEvents(ctx context.Context, roomID int64, limit int) ([]domain.RoomEvent, error)

	GetRoomsByServerID(ctx context.Context, serverID int64) ([]domain.Room, error)
	GetRoom(ctx context.Context, roomID int64) (*domain.RoomInfo, error)
	GetRoomConfig(ctx context.Context, configID int64) (*domain.RoomConfig, error)
	GetCurrentRoundByRoomID(ctx context.Context, roomID int64) (*int64, error)
	GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error)
}
