package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	appcfg "game_server/internal/config"
	"game_server/internal/domain"
	"game_server/internal/repository"
	"game_server/pkg/storage"

	"github.com/sirupsen/logrus"
)

const defaultRNGDistributeURL = "http://rng-stub:7001/winnings/distribute"

type rngDistributeRequest struct {
	Probabilities []float64 `json:"probabilities"`
	WinnersCount  int       `json:"winners_count"`
}

type rngDistributeResponse struct {
	WinningPositions []int `json:"winning_positions"`
}

type RoomService struct {
	repo             repository.RoomRepository
	logger           *logrus.Logger
	serverID         int64
	timerService     *TimerService
	roomCache        *storage.RedisClient
	rngDistributeURL string
	httpClient       *http.Client
}

func NewRoomService(
	repo repository.RoomRepository,
	log *logrus.Logger,
	serverID int64,
	roomCache *storage.RedisClient,
	rngServiceURL string,
) *RoomService {
	if log == nil {
		log = logrus.New()
	}

	return &RoomService{
		repo:             repo,
		logger:           log,
		serverID:         serverID,
		roomCache:        roomCache,
		rngDistributeURL: normalizeRNGDistributeURL(rngServiceURL),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *RoomService) SetTimerService(timerService *TimerService) {
	s.timerService = timerService
}

// InitializeRoomsCache loads all rooms for this server and caches them with their configs.
func (s *RoomService) InitializeRoomsCache(ctx context.Context) error {
	rooms, err := s.repo.GetRoomsByServerID(ctx, s.serverID)
	if err != nil {
		return fmt.Errorf("get rooms by server id: %w", err)
	}

	if s.roomCache != nil {
		for _, room := range rooms {
			roomInfo, err := s.repo.GetRoom(ctx, room.RoomID)
			if err != nil {
				return fmt.Errorf("get room info for cache: %w", err)
			}
			if roomInfo == nil {
				continue
			}
			if err := s.cacheRoomInfo(ctx, roomInfo); err != nil {
				s.logger.WithError(err).Warnf("failed to cache room %d", room.RoomID)
			}
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"server_id": s.serverID,
		"count":     len(rooms),
	}).Info("Initialized rooms cache")

	return nil
}

func (s *RoomService) JoinRoom(ctx context.Context, userID, roomID int64) (int64, error) {
	return s.joinRoom(ctx, userID, roomID, nil)
}

func (s *RoomService) JoinRoomWithSeat(ctx context.Context, userID, roomID int64, numberInRoom int) (int64, error) {
	return s.joinRoom(ctx, userID, roomID, &numberInRoom)
}

func (s *RoomService) joinRoom(ctx context.Context, userID, roomID int64, requestedSeat *int) (int64, error) {
	roomInfo, err := s.resolveRoom(ctx, roomID)
	if err != nil {
		return 0, err
	}

	if requestedSeat != nil && (*requestedSeat < 1 || *requestedSeat > roomInfo.Config.Capacity) {
		return 0, repository.ErrInvalidSeatNumber
	}

	var participantID int64
	var roundID int64
	shouldStartTimer := false
	configTimeSeconds := roomInfo.Config.Time
	minUsers := roomInfo.Config.MinUsers

	err = s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		txRoomInfo, err := ts.GetRoomForUpdate(ctx, roomID)
		if err != nil {
			return err
		}
		if txRoomInfo.Room.ServerID != s.serverID {
			return repository.ErrWrongGameServer
		}

		roomConfig := txRoomInfo.Config
		if requestedSeat != nil && (*requestedSeat < 1 || *requestedSeat > roomConfig.Capacity) {
			return repository.ErrInvalidSeatNumber
		}

		if txRoomInfo.CurrentRoundID != nil && txRoomInfo.CurrentRoundStatus != nil {
			switch *txRoomInfo.CurrentRoundStatus {
			case "active":
				return repository.ErrGameAlreadyStarted
			case "finished", "cancelled":
				return repository.ErrRoundNotJoinable
			}
		}

		if txRoomInfo.CurrentRoundID != nil {
			roundID = *txRoomInfo.CurrentRoundID
			if txRoomInfo.ActiveParticipantsCount == 0 {
				shouldStartTimer = true
			}
		} else {
			roundID, err = ts.CreateRound(ctx, roomID)
			if err != nil {
				return fmt.Errorf("create round: %w", err)
			}
			if err := ts.UpdateRoundStatus(ctx, roundID, "waiting"); err != nil {
				return fmt.Errorf("set round status: %w", err)
			}
			shouldStartTimer = true
		}

		activeCount, err := ts.GetActiveParticipantsCount(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get active participants count: %w", err)
		}
		if activeCount >= roomConfig.Capacity {
			return repository.ErrRoomIsFull
		}

		userSeatsCount, err := ts.CountUserActiveParticipants(ctx, roundID, userID)
		if err != nil {
			return fmt.Errorf("count user active participants: %w", err)
		}
		if userSeatsCount >= maxSeatsPerUser(roomConfig.Capacity) {
			return repository.ErrMaxSeatsExceeded
		}

		balance, err := ts.GetBalanceLocked(ctx, userID)
		if err != nil {
			return fmt.Errorf("lock balance: %w", err)
		}
		if balance < roomConfig.RegistrationPrice {
			return repository.ErrInsufficientBalance
		}

		numberInRoom := 0
		if requestedSeat != nil {
			occupied, err := ts.IsSeatOccupied(ctx, roundID, *requestedSeat)
			if err != nil {
				return fmt.Errorf("check seat occupied: %w", err)
			}
			if occupied {
				return repository.ErrSeatAlreadyTaken
			}
			numberInRoom = *requestedSeat
		} else {
			numberInRoom, err = ts.FindFreeNumberInRoom(ctx, roundID, roomConfig.Capacity)
			if err != nil {
				if strings.Contains(err.Error(), "no free spots") {
					return repository.ErrRoomIsFull
				}
				return fmt.Errorf("find free spot: %w", err)
			}
		}

		pID, err := ts.CreateParticipant(ctx, userID, roundID, numberInRoom)
		if err != nil {
			return fmt.Errorf("create participant: %w", err)
		}

		expiresAt := time.Now().Add(time.Duration(roomConfig.Time)*time.Second + appcfg.ReservationGracePeriod)
		if _, err := ts.ReserveEntry(ctx, pID, roomConfig.RegistrationPrice, expiresAt); err != nil {
			return fmt.Errorf("reserve entry: %w", err)
		}

		if err := ts.UpdateBalance(ctx, userID, -roomConfig.RegistrationPrice); err != nil {
			return fmt.Errorf("deduct balance: %w", err)
		}

		participantID = pID
		configTimeSeconds = roomConfig.Time
		minUsers = roomConfig.MinUsers
		return nil
	})
	if err != nil {
		return 0, err
	}

	if shouldStartTimer && s.timerService != nil {
		s.timerService.StartTimer(context.Background(), roundID, roomID, minUsers, configTimeSeconds)
	}

	s.refreshRoomCache(ctx, roomID)
	return participantID, nil
}

// PurchaseBoost atomically purchases the configured boost once for the user in a round.
func (s *RoomService) PurchaseBoost(ctx context.Context, participantID, userID int64, boostPower, boostCost int64) error {
	if boostPower <= 0 {
		return repository.ErrInvalidAmount
	}
	if boostCost <= 0 {
		return errors.New("missing boost cost")
	}

	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		participant, err := ts.GetParticipantByID(ctx, participantID)
		if err != nil {
			return err
		}
		if participant.UserID != userID {
			return repository.ErrParticipantAccessDenied
		}
		if participant.ExitRoomAt != nil {
			return repository.ErrParticipantNotFound
		}

		status, err := ts.GetRoundStatus(ctx, participant.RoundsID)
		if err != nil {
			return fmt.Errorf("get round status: %w", err)
		}
		if !canChangeBoost(status) {
			return repository.ErrGameAlreadyStarted
		}

		participants, err := ts.GetParticipantsByRoundID(ctx, participant.RoundsID)
		if err != nil {
			return fmt.Errorf("get participants by round: %w", err)
		}
		for _, roundParticipant := range participants {
			if roundParticipant.UserID == userID && roundParticipant.Boost > 0 {
				return repository.ErrBoostAlreadyPurchased
			}
		}

		balance, err := ts.GetBalanceLocked(ctx, userID)
		if err != nil {
			return fmt.Errorf("lock balance: %w", err)
		}

		if balance < boostCost {
			return repository.ErrInsufficientBalance
		}

		expiresAt := time.Now().Add(appcfg.ReservationGracePeriod)
		if _, err := ts.ReserveBoost(ctx, participantID, boostCost, expiresAt); err != nil {
			return fmt.Errorf("reserve boost: %w", err)
		}
		if err := ts.UpdateBalance(ctx, userID, -boostCost); err != nil {
			return fmt.Errorf("deduct boost cost: %w", err)
		}

		return ts.UpdateParticipantBoost(ctx, participantID, int(boostPower))
	})
}

// CancelBoost atomically refunds the current boost reservation and resets the participant boost.
func (s *RoomService) CancelBoost(ctx context.Context, participantID, userID int64) error {
	return s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		participant, err := ts.GetParticipantByID(ctx, participantID)
		if err != nil {
			return err
		}
		if participant.UserID != userID {
			return repository.ErrParticipantAccessDenied
		}
		if participant.ExitRoomAt != nil {
			return nil
		}

		status, err := ts.GetRoundStatus(ctx, participant.RoundsID)
		if err != nil {
			return fmt.Errorf("get round status: %w", err)
		}
		if !canChangeBoost(status) {
			return repository.ErrGameAlreadyStarted
		}

		refund, err := ts.ReleaseBoostReservations(ctx, participantID)
		if err != nil && !errors.Is(err, repository.ErrActiveReservationNotFound) {
			return fmt.Errorf("release boost reservation: %w", err)
		}
		if refund > 0 {
			if err := ts.UpdateBalance(ctx, userID, refund); err != nil {
				return fmt.Errorf("refund boost: %w", err)
			}
		}

		return ts.UpdateParticipantBoost(ctx, participantID, 0)
	})
}

func canChangeBoost(roundStatus string) bool {
	return roundStatus == "waiting"
}

// LeaveRoom atomically releases all reservations, refunds the participant, and frees the seat.
func (s *RoomService) LeaveRoom(ctx context.Context, participantID, userID int64) error {
	var roundID int64
	roundCancelled := false
	roomID := int64(0)

	err := s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		participant, err := ts.GetParticipantByID(ctx, participantID)
		if err != nil {
			return err
		}
		if participant.UserID != userID {
			return repository.ErrParticipantAccessDenied
		}
		if participant.ExitRoomAt != nil {
			return nil
		}

		roundID = participant.RoundsID
		roundInfo, err := ts.GetRoundInfo(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get round info: %w", err)
		}
		roomID = roundInfo.RoomID
		if roundInfo.Status == "active" {
			return repository.ErrGameAlreadyStarted
		}

		refund, err := ts.ReleaseAllReservations(ctx, participantID)
		if err != nil {
			return fmt.Errorf("release all reservations: %w", err)
		}
		if refund > 0 {
			if err := ts.UpdateBalance(ctx, userID, refund); err != nil {
				return fmt.Errorf("refund balance: %w", err)
			}
		}

		if err := ts.MarkParticipantExited(ctx, participantID); err != nil {
			return fmt.Errorf("mark participant exited: %w", err)
		}

		activeCount, err := ts.GetActiveParticipantsCount(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get active participants count: %w", err)
		}
		if activeCount == 0 {
			if err := ts.UpdateRoundStatus(ctx, roundID, "cancelled"); err != nil {
				return fmt.Errorf("set round status cancelled: %w", err)
			}
			if err := ts.ArchiveRound(ctx, roundID); err != nil {
				return fmt.Errorf("archive round: %w", err)
			}
			roundCancelled = true
		}

		return nil
	})
	if err != nil {
		return err
	}

	if roundCancelled && s.timerService != nil {
		s.timerService.StopTimer(roundID)
	}
	if roomID != 0 {
		s.refreshRoomCache(ctx, roomID)
	}
	return nil
}

// LeaveRoomByUser releases all active seats owned by the user in the room.
func (s *RoomService) LeaveRoomByUser(ctx context.Context, userID, roomID int64) (int64, error) {
	var roundID int64
	totalRefund := int64(0)
	roundCancelled := false

	err := s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		roomInfo, err := ts.GetRoomForUpdate(ctx, roomID)
		if err != nil {
			return err
		}
		if roomInfo == nil || roomInfo.Room == nil {
			return repository.ErrRoomNotFound
		}
		if roomInfo.Room.ServerID != s.serverID {
			return repository.ErrWrongGameServer
		}
		if roomInfo.CurrentRoundID == nil {
			return nil
		}

		roundID = *roomInfo.CurrentRoundID
		roundInfo, err := ts.GetRoundInfo(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get round info: %w", err)
		}
		if roundInfo.Status == "active" {
			return repository.ErrGameAlreadyStarted
		}
		if roundInfo.Status == "finished" || roundInfo.Status == "cancelled" {
			return repository.ErrRoundNotJoinable
		}

		participants, err := ts.GetParticipantsByRoundID(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get participants by round: %w", err)
		}

		leavingParticipants := make([]domain.RoundParticipant, 0)
		for _, participant := range participants {
			if participant.UserID == userID && participant.ExitRoomAt == nil {
				leavingParticipants = append(leavingParticipants, participant)
			}
		}
		if len(leavingParticipants) == 0 {
			return nil
		}

		for _, participant := range leavingParticipants {
			refund, err := ts.ReleaseAllReservations(ctx, participant.RoundParticipantID)
			if err != nil {
				return fmt.Errorf("release all reservations: %w", err)
			}
			if refund > 0 {
				if err := ts.UpdateBalance(ctx, userID, refund); err != nil {
					return fmt.Errorf("refund balance: %w", err)
				}
				totalRefund += refund
			}

			if err := ts.MarkParticipantExited(ctx, participant.RoundParticipantID); err != nil {
				return fmt.Errorf("mark participant exited: %w", err)
			}
		}

		activeCount, err := ts.GetActiveParticipantsCount(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get active participants count: %w", err)
		}
		if activeCount == 0 {
			if err := ts.UpdateRoundStatus(ctx, roundID, "cancelled"); err != nil {
				return fmt.Errorf("set round status cancelled: %w", err)
			}
			if err := ts.ArchiveRound(ctx, roundID); err != nil {
				return fmt.Errorf("archive round: %w", err)
			}
			roundCancelled = true
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	if roundCancelled && s.timerService != nil {
		s.timerService.StopTimer(roundID)
	}
	s.refreshRoomCache(ctx, roomID)
	return totalRefund, nil
}

// CancelWaitingRound releases every waiting participant reservation and archives the round.
func (s *RoomService) CancelWaitingRound(ctx context.Context, roundID int64) error {
	var roomID int64

	err := s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		roundInfo, err := ts.GetRoundInfo(ctx, roundID)
		if err != nil {
			return err
		}
		roomID = roundInfo.RoomID

		if roundInfo.Status == "finished" || roundInfo.Status == "cancelled" {
			return nil
		}
		if roundInfo.Status == "active" {
			return repository.ErrGameAlreadyStarted
		}

		participants, err := ts.GetParticipantsByRoundID(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get participants by round: %w", err)
		}

		for _, participant := range participants {
			refund, err := ts.ReleaseAllReservations(ctx, participant.RoundParticipantID)
			if err != nil {
				return fmt.Errorf("release reservations for participant %d: %w", participant.RoundParticipantID, err)
			}
			if refund > 0 {
				if err := ts.UpdateBalance(ctx, participant.UserID, refund); err != nil {
					return fmt.Errorf("refund participant %d: %w", participant.RoundParticipantID, err)
				}
			}
			if err := ts.MarkParticipantExited(ctx, participant.RoundParticipantID); err != nil {
				return fmt.Errorf("mark participant %d exited: %w", participant.RoundParticipantID, err)
			}
		}

		if err := ts.UpdateRoundStatus(ctx, roundID, "cancelled"); err != nil {
			return fmt.Errorf("set round status cancelled: %w", err)
		}
		if err := ts.ArchiveRound(ctx, roundID); err != nil {
			return fmt.Errorf("archive round: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if s.timerService != nil {
		s.timerService.StopTimer(roundID)
	}
	if roomID != 0 {
		s.refreshRoomCache(ctx, roomID)
	}
	return nil
}

// FinalizeRound commits active reservations, credits winners, archives the completed round, and creates the next round.
func (s *RoomService) FinalizeRound(ctx context.Context, roundID int64, winners map[int64]int64) error {
	_, err := s.finalizeRoundAndCreateNext(ctx, roundID, winners)
	if err != nil {
		return err
	}
	if s.timerService != nil {
		s.timerService.StopTimer(roundID)
	}
	return nil
}

func (s *RoomService) finalizeRoundAndCreateNext(ctx context.Context, roundID int64, winners map[int64]int64) (int64, error) {
	nextRoundID := int64(0)
	roomID := int64(0)

	err := s.repo.InTransaction(ctx, func(ts repository.TransactionScope) error {
		roundInfo, err := ts.GetRoundInfo(ctx, roundID)
		if err != nil {
			return err
		}
		roomID = roundInfo.RoomID

		switch roundInfo.Status {
		case "finished", "cancelled":
			return repository.ErrRoundAlreadyFinalized
		case "active":
		default:
			return fmt.Errorf("round is not active (status: %s)", roundInfo.Status)
		}

		participants, err := ts.GetParticipantsByRoundID(ctx, roundID)
		if err != nil {
			return fmt.Errorf("get participants by round: %w", err)
		}

		activeParticipants := make(map[int64]domain.RoundParticipant, len(participants))
		for _, participant := range participants {
			activeParticipants[participant.RoundParticipantID] = participant
		}

		for participantID := range winners {
			if _, ok := activeParticipants[participantID]; !ok {
				return fmt.Errorf("winner participant %d is not active in round %d", participantID, roundID)
			}
		}

		for _, participant := range participants {
			if _, err := ts.CommitReservations(ctx, participant.RoundParticipantID); err != nil && !errors.Is(err, repository.ErrActiveReservationNotFound) {
				return fmt.Errorf("commit reservations for participant %d: %w", participant.RoundParticipantID, err)
			}

			winAmount := winners[participant.RoundParticipantID]
			if winAmount > 0 {
				if err := ts.UpdateBalance(ctx, participant.UserID, winAmount); err != nil {
					return fmt.Errorf("credit winner %d: %w", participant.RoundParticipantID, err)
				}
				if err := ts.UpdateWinningMoney(ctx, participant.RoundParticipantID, winAmount); err != nil {
					return fmt.Errorf("update winning money for participant %d: %w", participant.RoundParticipantID, err)
				}
			}
		}

		if err := ts.UpdateRoundStatus(ctx, roundID, "finished"); err != nil {
			return fmt.Errorf("set round status finished: %w", err)
		}
		if err := ts.ArchiveRound(ctx, roundID); err != nil {
			return fmt.Errorf("archive round: %w", err)
		}

		nextRoundID, err = ts.CreateRound(ctx, roomID)
		if err != nil {
			return fmt.Errorf("create next round: %w", err)
		}
		if err := ts.UpdateRoundStatus(ctx, nextRoundID, "waiting"); err != nil {
			return fmt.Errorf("set next round status waiting: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	s.refreshRoomCache(ctx, roomID)
	return nextRoundID, nil
}

// GetRoomInfo returns the current room information.
func (s *RoomService) GetRoomInfo(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	return s.repo.GetRoom(ctx, roomID)
}

// GetParticipantInfo returns information about a single participant.
func (s *RoomService) GetParticipantInfo(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	return s.repo.GetParticipantByID(ctx, participantID)
}

// GetParticipantsByRound returns active participants of the round.
func (s *RoomService) GetParticipantsByRound(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	return s.repo.GetParticipantsByRoundID(ctx, roundID)
}

// GetRoundInfo returns the round metadata.
func (s *RoomService) GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error) {
	return s.repo.GetRoundInfo(ctx, roundID)
}

// GetRoomInfoByRound returns room information for the round's room.
func (s *RoomService) GetRoomInfoByRound(ctx context.Context, roundID int64) (*domain.RoomInfo, error) {
	roundInfo, err := s.repo.GetRoundInfo(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("get round info: %w", err)
	}
	return s.repo.GetRoom(ctx, roundInfo.RoomID)
}

// StartGameRound moves the waiting round into active state.
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

// FinalizeGameRound picks winners via RNG, finalizes balances, and returns the winning participants.
func (s *RoomService) FinalizeGameRound(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	roomInfo, err := s.GetRoomInfoByRound(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if roomInfo == nil || roomInfo.Config == nil {
		return nil, repository.ErrRoomNotFound
	}

	participants, err := s.repo.GetParticipantsByRoundID(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("get participants by round: %w", err)
	}
	if len(participants) == 0 {
		return nil, errors.New("round has no active participants")
	}

	winningPositions, err := s.requestWinningPositions(ctx, roomInfo.Config, participants)
	if err != nil {
		return nil, err
	}

	payouts := buildPayouts(roomInfo.Config, len(participants), len(winningPositions))
	payoutsByParticipant := make(map[int64]int64, len(winningPositions))
	winners := make([]domain.RoundParticipant, 0, len(winningPositions))
	participantsBySeat := make(map[int]domain.RoundParticipant, len(participants))
	for _, participant := range participants {
		participantsBySeat[participant.NumberInRoom] = participant
	}

	for idx, winningPosition := range winningPositions {
		participant, ok := participantsBySeat[winningPosition]
		if !ok {
			return nil, fmt.Errorf("winning position %d does not have an active participant", winningPosition)
		}
		participant.WinningMoney = payouts[idx]
		payoutsByParticipant[participant.RoundParticipantID] = payouts[idx]
		winners = append(winners, participant)
	}

	if _, err := s.finalizeRoundAndCreateNext(ctx, roundID, payoutsByParticipant); err != nil {
		return nil, err
	}
	if s.timerService != nil {
		s.timerService.StopTimer(roundID)
	}

	return winners, nil
}

func (s *RoomService) resolveRoom(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	if s.roomCache != nil {
		cachedRoom := domain.RoomCacheData{}
		if err := s.roomCache.Get(ctx, fmt.Sprintf(appcfg.RedisKeyRoom, roomID), &cachedRoom); err == nil {
			if cachedRoom.ServerID != s.serverID {
				return nil, repository.ErrWrongGameServer
			}
		}
	}

	roomInfo, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if roomInfo == nil {
		return nil, repository.ErrRoomNotFound
	}
	if roomInfo.Room.ServerID != s.serverID {
		return nil, repository.ErrWrongGameServer
	}

	s.refreshRoomCache(ctx, roomID)
	return roomInfo, nil
}

func (s *RoomService) refreshRoomCache(ctx context.Context, roomID int64) {
	if s.roomCache == nil {
		return
	}

	roomInfo, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		s.logger.WithError(err).Warnf("failed to refresh room cache for room %d", roomID)
		return
	}
	if roomInfo == nil {
		return
	}

	if err := s.cacheRoomInfo(ctx, roomInfo); err != nil {
		s.logger.WithError(err).Warnf("failed to cache room %d", roomID)
	}
}

func (s *RoomService) cacheRoomInfo(ctx context.Context, roomInfo *domain.RoomInfo) error {
	if s.roomCache == nil || roomInfo == nil || roomInfo.Room == nil || roomInfo.Config == nil {
		return nil
	}

	cacheData := domain.RoomCacheData{
		RoomID:         roomInfo.Room.RoomID,
		ConfigID:       roomInfo.Room.ConfigID,
		ServerID:       roomInfo.Room.ServerID,
		CurrentRoundID: roomInfo.CurrentRoundID,
		Status:         roomInfo.Room.Status,
	}
	if err := s.roomCache.Set(ctx, fmt.Sprintf(appcfg.RedisKeyRoom, roomInfo.Room.RoomID), cacheData, appcfg.RedisRoomCacheTTL); err != nil {
		return fmt.Errorf("cache room: %w", err)
	}

	configData := domain.RoomConfigCacheData{
		ConfigID:            roomInfo.Config.ConfigID,
		GameID:              roomInfo.Config.GameID,
		Capacity:            roomInfo.Config.Capacity,
		RegistrationPrice:   roomInfo.Config.RegistrationPrice,
		IsBoost:             roomInfo.Config.IsBoost,
		BoostPrice:          roomInfo.Config.BoostPrice,
		BoostPower:          roomInfo.Config.BoostPower,
		NumberWinners:       roomInfo.Config.NumberWinners,
		WinningDistribution: roomInfo.Config.WinningDistribution,
		Commission:          roomInfo.Config.Commission,
		Time:                roomInfo.Config.Time,
		MinUsers:            roomInfo.Config.MinUsers,
	}
	if err := s.roomCache.Set(ctx, fmt.Sprintf(appcfg.RedisKeyRoomConfig, roomInfo.Config.ConfigID), configData, appcfg.RedisRoomCacheTTL); err != nil {
		return fmt.Errorf("cache room config: %w", err)
	}

	return s.roomCache.SAdd(ctx, fmt.Sprintf(appcfg.RedisKeyGameServerRooms, s.serverID), roomInfo.Room.RoomID)
}

func (s *RoomService) requestWinningPositions(ctx context.Context, config *domain.RoomConfig, participants []domain.RoundParticipant) ([]int, error) {
	winnersCount := config.NumberWinners
	countPlayers := config.Capacity
	boostPower := config.BoostPower

	if winnersCount > countPlayers {
		winnersCount = countPlayers
	}

	sumProbabilities := 0.0
	probabilities := make([]float64, countPlayers)
	for _, participant := range participants {
		weight := 1.0 / float64(winnersCount)
		if participant.Boost > 0 {
			weight += float64(boostPower) / 100.0
		}
		probabilities[participant.NumberInRoom-1] = weight
		sumProbabilities += weight
	}

	for _, participant := range participants {
		probabilities[participant.NumberInRoom-1] /= sumProbabilities
	}

	payload := rngDistributeRequest{
		Probabilities: probabilities,
		WinnersCount:  winnersCount,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal RNG payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.rngDistributeURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build RNG request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call RNG service: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RNG service returned status %d", response.StatusCode)
	}

	var rngResponse rngDistributeResponse
	if err := json.NewDecoder(response.Body).Decode(&rngResponse); err != nil {
		return nil, fmt.Errorf("decode RNG response: %w", err)
	}
	if len(rngResponse.WinningPositions) != winnersCount {
		return nil, fmt.Errorf("RNG service returned %d winners, expected %d", len(rngResponse.WinningPositions), winnersCount)
	}

	return rngResponse.WinningPositions, nil
}

func buildPayouts(config *domain.RoomConfig, participantsCount int, winnersCount int) []int64 {
	if winnersCount == 0 {
		return nil
	}

	grossBank := config.RegistrationPrice * int64(participantsCount)
	commission := grossBank * int64(config.Commission) / 100
	prizePool := grossBank - commission
	payouts := make([]int64, winnersCount)
	distribution := config.WinningDistribution
	if len(distribution) > winnersCount {
		distribution = distribution[:winnersCount]
	}

	assigned := int64(0)
	for idx := 0; idx < winnersCount; idx++ {
		if idx == winnersCount-1 {
			payouts[idx] = prizePool - assigned
			continue
		}

		share := 0
		if idx < len(distribution) {
			share = distribution[idx]
		}
		payouts[idx] = prizePool * int64(share) / 100
		assigned += payouts[idx]
	}

	return payouts
}

func maxSeatsPerUser(capacity int) int {
	limit := capacity / 2
	if limit < 1 {
		return 1
	}
	return limit
}

func normalizeRNGDistributeURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return defaultRNGDistributeURL
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return defaultRNGDistributeURL
	}

	switch parsed.Path {
	case "", "/":
		parsed.Path = "/winnings/distribute"
	case "/random":
		parsed.Path = "/winnings/distribute"
	}

	return parsed.String()
}
