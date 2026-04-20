package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"game_server/internal/repository"
)

type RoomService struct {
	repo   repository.RoomRepository
	logger *logrus.Logger
}

func NewRoomService(repo repository.RoomRepository, log *logrus.Logger) *RoomService {
	return &RoomService{repo: repo, logger: log}
}

// JoinRoom атомарно: блокирует баланс, создаёт участника (1 место), резервирует вход, списывает деньги.
// capacity и currentActivePlayers передаются из game_server для валидации без гонки.
func (s *RoomService) JoinRoom(ctx context.Context, userID, roundID, entryPrice int64, capacity, currentActivePlayers int, expiresAt time.Time) (int64, error) {
	if currentActivePlayers >= capacity {
		return 0, repository.ErrRoomIsFull
	}

	var participantID int64
	err := s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		bal, err := ts.GetBalanceLocked(ctx, userID)
		if err != nil {
			return fmt.Errorf("lock balance: %w", err)
		}
		if bal < entryPrice {
			return repository.ErrInsufficientBalance
		}

		pID, err := ts.CreateParticipant(ctx, userID, roundID, 1)
		if err != nil {
			return fmt.Errorf("create participant: %w", err)
		}

		if _, err = ts.ReserveEntry(ctx, pID, entryPrice, expiresAt); err != nil {
			return fmt.Errorf("reserve entry: %w", err)
		}

		if err = ts.UpdateBalance(ctx, userID, -entryPrice); err != nil {
			return fmt.Errorf("deduct balance: %w", err)
		}

		participantID = pID
		return nil
	})
	return participantID, err
}

// PurchaseBoost атомарно: отменяет старый буст, создаёт новый резерв, обновляет boost в БД, списывает стоимость.
func (s *RoomService) PurchaseBoost(ctx context.Context, participantID, userID int64, boostValue, boostCost int64, expiresAt time.Time) error {
	if boostValue > 0 && boostCost <= 0 {
		return errors.New("missing boost cost")
	}

	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		bal, err := ts.GetBalanceLocked(ctx, userID)
		if err != nil {
			return fmt.Errorf("lock balance: %w", err)
		}

		// Возвращаем старый резерв буста (если был)
		oldRefund, _ := ts.ReleaseBoostReservations(ctx, participantID)
		if oldRefund > 0 {
			if err = ts.UpdateBalance(ctx, userID, oldRefund); err != nil {
				return fmt.Errorf("refund old boost: %w", err)
			}
		}

		available := bal + oldRefund
		if available < boostCost {
			return repository.ErrInsufficientBalance
		}

		if _, err = ts.ReserveBoost(ctx, participantID, boostCost, expiresAt); err != nil {
			return fmt.Errorf("reserve boost: %w", err)
		}
		if err = ts.UpdateBalance(ctx, userID, -boostCost); err != nil {
			return fmt.Errorf("deduct boost cost: %w", err)
		}

		return ts.UpdateParticipantBoost(ctx, participantID, int(boostValue))
	})
}

// CancelBoost атомарно: сбрасывает boost=0, отменяет резервы буста, возвращает деньги.
func (s *RoomService) CancelBoost(ctx context.Context, participantID, userID int64) error {
	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		refund, err := ts.ReleaseBoostReservations(ctx, participantID)
		if err != nil && !errors.Is(err, repository.ErrActiveReservationNotFound) {
			return fmt.Errorf("release boost: %w", err)
		}
		if refund > 0 {
			if err = ts.UpdateBalance(ctx, userID, refund); err != nil {
				return fmt.Errorf("refund boost: %w", err)
			}
		}
		return ts.UpdateParticipantBoost(ctx, participantID, 0)
	})
}

// LeaveRoom атомарно: отменяет ВСЕ резервы (вход + буст), возвращает деньги, помечает участника как вышедшего.
func (s *RoomService) LeaveRoom(ctx context.Context, participantID, userID int64) error {
	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		refund, err := ts.ReleaseAllReservations(ctx, participantID)
		if err != nil {
			return fmt.Errorf("release all: %w", err)
		}
		if refund > 0 {
			if err = ts.UpdateBalance(ctx, userID, refund); err != nil {
				return fmt.Errorf("refund: %w", err)
			}
		}
		return ts.MarkParticipantExited(ctx, participantID)
	})
}

// FinalizeRound атомарно: подтверждает резервы всех участников, начисляет выигрыши, архивирует раунд.
// winners: map[participantID]winAmount (0 для не-победителей)
func (s *RoomService) FinalizeRound(ctx context.Context, roundID int64, winners map[int64]int64) error {
	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		for pID, winAmt := range winners {
			if _, err := ts.CommitReservations(ctx, pID); err != nil && !errors.Is(err, repository.ErrActiveReservationNotFound) {
				return fmt.Errorf("commit reservations for %d: %w", pID, err)
			}

			if winAmt > 0 {
				userID, err := ts.GetParticipantUserID(ctx, pID)
				if err != nil {
					return fmt.Errorf("get user id for %d: %w", pID, err)
				}
				if err = ts.UpdateBalance(ctx, userID, winAmt); err != nil {
					return fmt.Errorf("credit win for %d: %w", pID, err)
				}
				if err = ts.UpdateWinningMoney(ctx, pID, winAmt); err != nil {
					return fmt.Errorf("update winning money for %d: %w", pID, err)
				}
			}
		}
		return ts.ArchiveRound(ctx, roundID)
	})
}
