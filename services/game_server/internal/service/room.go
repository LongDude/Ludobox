package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	appcfg "game_server/internal/config"
	"game_server/internal/repository"
)

type RoomService struct {
	repo     repository.RoomRepository
	logger   *logrus.Logger
	serverID int64
}

func NewRoomService(repo repository.RoomRepository, log *logrus.Logger, serverID int64) *RoomService {
	return &RoomService{repo: repo, logger: log, serverID: serverID}
}

// InitializeRoomsCache loads all rooms for this server and caches them with their configs
// This should be called at startup to warm up the cache
func (s *RoomService) InitializeRoomsCache(ctx context.Context) error {
	rooms, err := s.repo.GetRoomsByServerID(ctx, s.serverID)
	if err != nil {
		return fmt.Errorf("get rooms by server id: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"server_id": s.serverID,
		"count":     len(rooms),
	}).Info("Initialized rooms cache")

	// TODO: Cache rooms and configs in Redis
	// This will be implemented when Redis cache layer is added

	return nil
}

// JoinRoom атомарно: блокирует баланс, создаёт участника, резервирует вход, списывает деньги.
// Автоматически создает раунд если его нет, занимает первое свободное место.
func (s *RoomService) JoinRoom(ctx context.Context, userID, roomID int64) (int64, error) {
	roomInfo, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return 0, fmt.Errorf("get room: %w", err)
	}

	if roomInfo == nil {
		return 0, errors.New("room not found")
	}

	roomConfig := roomInfo.Config
	activeCount := roomInfo.ActiveParticipantsCount
	capacity := roomConfig.Capacity

	if activeCount >= capacity {
		return 0, repository.ErrRoomIsFull
	}

	entryPrice := roomConfig.RegistrationPrice
	// expiresAt = createdAt + time (из конфига комнаты) + 600
	expiresAt := time.Now().Add(time.Duration(roomConfig.Time)*time.Second + appcfg.ReservationGracePeriod)

	var participantID int64
	err = s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		// Получаем или создаём текущий раунд
		var roundID int64
		if roomInfo.CurrentRoundID != nil {
			roundID = *roomInfo.CurrentRoundID
		} else {
			// Создаём новый раунд если его нет
			newRoundID, err := ts.CreateRound(ctx, roomID)
			if err != nil {
				return fmt.Errorf("create round: %w", err)
			}
			roundID = newRoundID
			// Устанавливаем статус в 'waiting'
			if err := ts.UpdateRoundStatus(ctx, roundID, "waiting"); err != nil {
				return fmt.Errorf("set round status: %w", err)
			}
		}

		// Проверяем баланс
		bal, err := ts.GetBalanceLocked(ctx, userID)
		if err != nil {
			return fmt.Errorf("lock balance: %w", err)
		}
		if bal < entryPrice {
			return repository.ErrInsufficientBalance
		}

		// Находим свободное место в комнате
		numberInRoom, err := ts.FindFreeNumberInRoom(ctx, roundID, capacity)
		if err != nil {
			return fmt.Errorf("find free spot: %w", err)
		}

		// Создаём участника
		pID, err := ts.CreateParticipant(ctx, userID, roundID, numberInRoom)
		if err != nil {
			return fmt.Errorf("create participant: %w", err)
		}

		// Резервируем вход
		if _, err = ts.ReserveEntry(ctx, pID, entryPrice, expiresAt); err != nil {
			return fmt.Errorf("reserve entry: %w", err)
		}

		// Списываем деньги
		if err = ts.UpdateBalance(ctx, userID, -entryPrice); err != nil {
			return fmt.Errorf("deduct balance: %w", err)
		}

		participantID = pID
		return nil
	})
	return participantID, err
}

// PurchaseBoost атомарно: отменяет старый буст, создаёт новый резерв, обновляет boost в БД, списывает стоимость.
func (s *RoomService) PurchaseBoost(ctx context.Context, participantID, userID int64, boostValue, boostCost int64) error {
	if boostValue > 0 && boostCost <= 0 {
		return errors.New("missing boost cost")
	}

	expiresAt := time.Now().Add(appcfg.ReservationGracePeriod)

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
// Если это был последний участник - удаляет раунд.
func (s *RoomService) LeaveRoom(ctx context.Context, participantID, userID int64) error {
	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		// Получаем информацию о участнике
		participant, err := s.repo.GetParticipantByID(ctx, participantID)
		if err != nil {
			return fmt.Errorf("get participant: %w", err)
		}

		roundID := participant.RoundsID

		// Отменяем все резервы и возвращаем деньги
		refund, err := ts.ReleaseAllReservations(ctx, participantID)
		if err != nil {
			return fmt.Errorf("release all: %w", err)
		}
		if refund > 0 {
			if err = ts.UpdateBalance(ctx, userID, refund); err != nil {
				return fmt.Errorf("refund: %w", err)
			}
		}

		// Помечаем участника как вышедшего
		if err = ts.MarkParticipantExited(ctx, participantID); err != nil {
			return fmt.Errorf("mark exited: %w", err)
		}

		// Проверяем количество активных участников
		activeCount, err := ts.GetActiveParticipantsCount(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get active participants count: %w", err)
		}

		// Если нет активных участников - удаляем раунд (мягкое удаление)
		if activeCount == 0 {
			if err = ts.UpdateRoundStatus(ctx, roundID, "cancelled"); err != nil {
				return fmt.Errorf("cancel round: %w", err)
			}
			if err = ts.ArchiveRound(ctx, roundID); err != nil {
				return fmt.Errorf("archive round: %w", err)
			}
		}

		return nil
	})
}

// FinalizeRound атомарно: подтверждает резервы всех участников, начисляет выигрыши, архивирует раунд.
// winners: map[participantID]winAmount (0 для не-победителей)
// Проверяет, что раунд ещё активный (не заканчивался)
func (s *RoomService) FinalizeRound(ctx context.Context, roundID int64, winners map[int64]int64) error {
	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		// Проверяем статус раунда - должен быть 'active'
		status, err := ts.GetRoundStatus(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get round status: %w", err)
		}
		if status != "active" {
			return fmt.Errorf("round is not active (status: %s)", status)
		}

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

		// Устанавливаем статус в 'finished' и архивируем
		if err := ts.UpdateRoundStatus(ctx, roundID, "finished"); err != nil {
			return fmt.Errorf("set round status finished: %w", err)
		}
		return ts.ArchiveRound(ctx, roundID)
	})
}

// GetRoomInfo возвращает информацию о комнате
func (s *RoomService) GetRoomInfo(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	return s.repo.GetRoom(ctx, roomID)
}

// GetParticipantInfo возвращает информацию об участнике
func (s *RoomService) GetParticipantInfo(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	return s.repo.GetParticipantByID(ctx, participantID)
}

// GetRoomInfoByRound возвращает информацию о комнате по раунду
func (s *RoomService) GetRoomInfoByRound(ctx context.Context, roundID int64) (*domain.RoomInfo, error) {
	roundInfo, err := s.repo.GetRoundInfo(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("get round info: %w", err)
	}
	return s.repo.GetRoom(ctx, roundInfo.RoomID)
}

// StartGameRound переводит раунд в статус 'active'
func (s *RoomService) StartGameRound(ctx context.Context, roundID int64) error {
	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		status, err := ts.GetRoundStatus(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get round status: %w", err)
		}
		if status != "waiting" {
			return fmt.Errorf("round is not in waiting state (status: %s)", status)
		}
		return ts.UpdateRoundStatus(ctx, roundID, "active")
	})
}

// FinalizeGameRound финализирует раунд (определяет победителей, выплачивает деньги)
func (s *RoomService) FinalizeGameRound(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	// TODO: Реализовать вызов RNG сервиса, определение победителей и выплату
	return nil, errors.New("not implemented")
}
