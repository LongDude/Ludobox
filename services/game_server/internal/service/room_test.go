package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"game_server/internal/domain"
	"game_server/internal/repository"
)

type reservationData struct {
	Type   string
	Amount int64
	Status string
}

type mockTransactionScope struct {
	roomInfo          *domain.RoomInfo
	rounds            map[int64]*domain.Round
	participants      map[int64]*domain.RoundParticipant
	reservations      map[int64][]*reservationData
	balances          map[int64]int64
	nextParticipantID int64
	nextRoundID       int64
}

func newMockTransactionScope() *mockTransactionScope {
	roundID := int64(1)
	status := "waiting"

	return &mockTransactionScope{
		roomInfo: &domain.RoomInfo{
			Room: &domain.Room{
				RoomID:         1,
				ConfigID:       1,
				ServerID:       1,
				Status:         domain.RoomStatusOpen,
				CurrentPlayers: 0,
			},
			Config: &domain.RoomConfig{
				ConfigID:            1,
				Capacity:            4,
				RegistrationPrice:   100,
				IsBoost:             true,
				BoostPrice:          50,
				BoostPower:          25,
				NumberWinners:       2,
				WinningDistribution: []int{60, 40},
				Time:                60,
				MinUsers:            2,
			},
			CurrentRoundID:     &roundID,
			CurrentRoundStatus: &status,
		},
		rounds: map[int64]*domain.Round{
			roundID: {
				RoundsID:  roundID,
				RoomID:    1,
				Status:    "waiting",
				CreatedAt: time.Now(),
			},
		},
		participants:      make(map[int64]*domain.RoundParticipant),
		reservations:      make(map[int64][]*reservationData),
		balances:          map[int64]int64{100: 1000, 200: 1000},
		nextParticipantID: 1,
		nextRoundID:       2,
	}
}

func (m *mockTransactionScope) cloneRoomInfo() *domain.RoomInfo {
	roomCopy := *m.roomInfo.Room
	configCopy := *m.roomInfo.Config

	var roundID *int64
	if m.roomInfo.CurrentRoundID != nil {
		value := *m.roomInfo.CurrentRoundID
		roundID = &value
	}

	var roundStatus *string
	if m.roomInfo.CurrentRoundStatus != nil {
		value := *m.roomInfo.CurrentRoundStatus
		roundStatus = &value
	}

	activeCount := 0
	if roundID != nil {
		activeCount, _ = m.GetActiveParticipantsCount(context.Background(), *roundID)
	}
	roomCopy.CurrentPlayers = activeCount

	return &domain.RoomInfo{
		Room:                    &roomCopy,
		Config:                  &configCopy,
		CurrentRoundID:          roundID,
		CurrentRoundStatus:      roundStatus,
		ActiveParticipantsCount: activeCount,
	}
}

func (m *mockTransactionScope) GetRoomForUpdate(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	if roomID != m.roomInfo.Room.RoomID {
		return nil, repository.ErrRoomNotFound
	}
	return m.cloneRoomInfo(), nil
}

func (m *mockTransactionScope) GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error) {
	round, ok := m.rounds[roundID]
	if !ok {
		return nil, repository.ErrRoundArchived
	}
	copyRound := *round
	return &copyRound, nil
}

func (m *mockTransactionScope) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	participant, ok := m.participants[participantID]
	if !ok {
		return nil, repository.ErrParticipantNotFound
	}
	copyParticipant := *participant
	return &copyParticipant, nil
}

func (m *mockTransactionScope) GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	participants := make([]domain.RoundParticipant, 0)
	for _, participant := range m.participants {
		if participant.RoundsID == roundID && participant.ExitRoomAt == nil {
			participants = append(participants, *participant)
		}
	}
	return participants, nil
}

func (m *mockTransactionScope) CountUserActiveParticipants(ctx context.Context, roundID, userID int64) (int, error) {
	count := 0
	for _, participant := range m.participants {
		if participant.RoundsID == roundID && participant.UserID == userID && participant.ExitRoomAt == nil {
			count++
		}
	}
	return count, nil
}

func (m *mockTransactionScope) IsSeatOccupied(ctx context.Context, roundID int64, numberInRoom int) (bool, error) {
	for _, participant := range m.participants {
		if participant.RoundsID == roundID && participant.NumberInRoom == numberInRoom && participant.ExitRoomAt == nil {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockTransactionScope) GetBalanceLocked(ctx context.Context, userID int64) (int64, error) {
	balance, ok := m.balances[userID]
	if !ok {
		return 0, errors.New("user not found")
	}
	return balance, nil
}

func (m *mockTransactionScope) UpdateBalance(ctx context.Context, userID int64, delta int64) error {
	m.balances[userID] += delta
	return nil
}

func (m *mockTransactionScope) CreateParticipant(ctx context.Context, userID, roundID int64, numberInRoom int) (int64, error) {
	id := m.nextParticipantID
	m.nextParticipantID++
	m.participants[id] = &domain.RoundParticipant{
		RoundParticipantID: id,
		UserID:             userID,
		RoundsID:           roundID,
		NumberInRoom:       numberInRoom,
	}
	return id, nil
}

func (m *mockTransactionScope) GetParticipantUserID(ctx context.Context, participantID int64) (int64, error) {
	participant, ok := m.participants[participantID]
	if !ok {
		return 0, repository.ErrParticipantNotFound
	}
	return participant.UserID, nil
}

func (m *mockTransactionScope) UpdateParticipantBoost(ctx context.Context, participantID int64, boost int) error {
	participant, ok := m.participants[participantID]
	if !ok {
		return repository.ErrParticipantNotFound
	}
	participant.Boost = boost
	return nil
}

func (m *mockTransactionScope) MarkParticipantExited(ctx context.Context, participantID int64) error {
	participant, ok := m.participants[participantID]
	if !ok {
		return repository.ErrParticipantNotFound
	}
	now := time.Now()
	participant.ExitRoomAt = &now
	return nil
}

func (m *mockTransactionScope) UpdateWinningMoney(ctx context.Context, participantID int64, amount int64) error {
	participant, ok := m.participants[participantID]
	if !ok {
		return repository.ErrParticipantNotFound
	}
	participant.WinningMoney = amount
	return nil
}

func (m *mockTransactionScope) ReserveEntry(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	m.reservations[participantID] = append(m.reservations[participantID], &reservationData{
		Type:   "entry_fee",
		Amount: amount,
		Status: "active",
	})
	return int64(len(m.reservations[participantID])), nil
}

func (m *mockTransactionScope) ReserveBoost(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	m.reservations[participantID] = append(m.reservations[participantID], &reservationData{
		Type:   "boost",
		Amount: amount,
		Status: "active",
	})
	return int64(len(m.reservations[participantID])), nil
}

func (m *mockTransactionScope) ReleaseAllReservations(ctx context.Context, participantID int64) (int64, error) {
	sum := int64(0)
	for _, reservation := range m.reservations[participantID] {
		if reservation.Status == "active" {
			sum += reservation.Amount
			reservation.Status = "released"
		}
	}
	return sum, nil
}

func (m *mockTransactionScope) ReleaseBoostReservations(ctx context.Context, participantID int64) (int64, error) {
	sum := int64(0)
	hasActive := false
	for _, reservation := range m.reservations[participantID] {
		if reservation.Type == "boost" && reservation.Status == "active" {
			sum += reservation.Amount
			reservation.Status = "released"
			hasActive = true
		}
	}
	if !hasActive {
		return 0, repository.ErrActiveReservationNotFound
	}
	return sum, nil
}

func (m *mockTransactionScope) CommitReservations(ctx context.Context, participantID int64) (int64, error) {
	sum := int64(0)
	hasActive := false
	for _, reservation := range m.reservations[participantID] {
		if reservation.Status == "active" {
			sum += reservation.Amount
			reservation.Status = "committed"
			hasActive = true
		}
	}
	if !hasActive {
		return 0, repository.ErrActiveReservationNotFound
	}
	return sum, nil
}

func (m *mockTransactionScope) ArchiveRound(ctx context.Context, roundID int64) error {
	round, ok := m.rounds[roundID]
	if !ok {
		return repository.ErrRoundArchived
	}
	now := time.Now()
	round.ArchivedAt = &now
	if m.roomInfo.CurrentRoundID != nil && *m.roomInfo.CurrentRoundID == roundID {
		m.roomInfo.CurrentRoundID = nil
		m.roomInfo.CurrentRoundStatus = nil
	}
	return nil
}

func (m *mockTransactionScope) CreateRound(ctx context.Context, roomID int64) (int64, error) {
	id := m.nextRoundID
	m.nextRoundID++
	m.rounds[id] = &domain.Round{
		RoundsID:  id,
		RoomID:    roomID,
		Status:    "waiting",
		CreatedAt: time.Now(),
	}
	m.roomInfo.CurrentRoundID = &id
	status := "waiting"
	m.roomInfo.CurrentRoundStatus = &status
	return id, nil
}

func (m *mockTransactionScope) UpdateRoundStatus(ctx context.Context, roundID int64, status string) error {
	round, ok := m.rounds[roundID]
	if !ok {
		return repository.ErrRoundArchived
	}
	round.Status = status
	if m.roomInfo.CurrentRoundID != nil && *m.roomInfo.CurrentRoundID == roundID {
		statusCopy := status
		m.roomInfo.CurrentRoundStatus = &statusCopy
	}
	return nil
}

func (m *mockTransactionScope) GetActiveParticipantsCount(ctx context.Context, roundID int64) (int, error) {
	count := 0
	for _, participant := range m.participants {
		if participant.RoundsID == roundID && participant.ExitRoomAt == nil {
			count++
		}
	}
	return count, nil
}

func (m *mockTransactionScope) FindFreeNumberInRoom(ctx context.Context, roundID int64, capacity int) (int, error) {
	for number := 1; number <= capacity; number++ {
		occupied, _ := m.IsSeatOccupied(ctx, roundID, number)
		if !occupied {
			return number, nil
		}
	}
	return 0, repository.ErrRoomIsFull
}

func (m *mockTransactionScope) GetRoundStatus(ctx context.Context, roundID int64) (string, error) {
	round, ok := m.rounds[roundID]
	if !ok {
		return "", repository.ErrRoundArchived
	}
	return round.Status, nil
}

type mockRoomRepository struct {
	scope *mockTransactionScope
}

func (m *mockRoomRepository) InTransaction(ctx context.Context, fn func(ts repository.TransactionScope) error) error {
	return fn(m.scope)
}

func (m *mockRoomRepository) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	return m.scope.GetParticipantByID(ctx, participantID)
}

func (m *mockRoomRepository) GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	return m.scope.GetParticipantsByRoundID(ctx, roundID)
}

func (m *mockRoomRepository) GetRoomsByServerID(ctx context.Context, serverID int64) ([]domain.Room, error) {
	if serverID != m.scope.roomInfo.Room.ServerID {
		return nil, nil
	}
	return []domain.Room{*m.scope.roomInfo.Room}, nil
}

func (m *mockRoomRepository) GetRoom(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	if roomID != m.scope.roomInfo.Room.RoomID {
		return nil, nil
	}
	return m.scope.cloneRoomInfo(), nil
}

func (m *mockRoomRepository) GetRoomConfig(ctx context.Context, configID int64) (*domain.RoomConfig, error) {
	if configID != m.scope.roomInfo.Config.ConfigID {
		return nil, repository.ErrRoomNotFound
	}
	configCopy := *m.scope.roomInfo.Config
	return &configCopy, nil
}

func (m *mockRoomRepository) GetCurrentRoundByRoomID(ctx context.Context, roomID int64) (*int64, error) {
	if roomID != m.scope.roomInfo.Room.RoomID {
		return nil, nil
	}
	return m.scope.roomInfo.CurrentRoundID, nil
}

func (m *mockRoomRepository) GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error) {
	return m.scope.GetRoundInfo(ctx, roundID)
}

func (m *mockRoomRepository) GetActiveParticipantsByRoomAndUser(ctx context.Context, roomID, userID int64) ([]domain.RoundParticipant, error) {
	if roomID != m.scope.roomInfo.Room.RoomID {
		return nil, nil
	}
	roundID := int64(0)
	if m.scope.roomInfo.CurrentRoundID != nil {
		roundID = *m.scope.roomInfo.CurrentRoundID
	}
	participants := make([]domain.RoundParticipant, 0)
	for _, participant := range m.scope.participants {
		if participant.RoundsID == roundID && participant.UserID == userID && participant.ExitRoomAt == nil {
			participants = append(participants, *participant)
		}
	}
	return participants, nil
}

func TestJoinRoomWithSeatSuccess(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	participantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 2)
	if err != nil {
		t.Fatalf("JoinRoomWithSeat failed: %v", err)
	}

	participant, err := repo.GetParticipantByID(ctx, participantID)
	if err != nil {
		t.Fatalf("GetParticipantByID failed: %v", err)
	}

	if participant.NumberInRoom != 2 {
		t.Fatalf("unexpected seat: got %d want 2", participant.NumberInRoom)
	}
	if scope.balances[100] != 900 {
		t.Fatalf("unexpected balance: got %d want 900", scope.balances[100])
	}
}

func TestJoinRoomRespectsHalfCapacityLimit(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	for _, seat := range []int{1, 2} {
		if _, err := service.JoinRoomWithSeat(ctx, 100, 1, seat); err != nil {
			t.Fatalf("preparing occupied seat %d failed: %v", seat, err)
		}
	}

	if _, err := service.JoinRoom(ctx, 100, 1); !errors.Is(err, repository.ErrMaxSeatsExceeded) {
		t.Fatalf("expected ErrMaxSeatsExceeded, got %v", err)
	}
}

func TestLeaveRoomRejectsForeignParticipant(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	participantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 1)
	if err != nil {
		t.Fatalf("JoinRoomWithSeat failed: %v", err)
	}

	if err := service.LeaveRoom(ctx, participantID, 200); !errors.Is(err, repository.ErrParticipantAccessDenied) {
		t.Fatalf("expected ErrParticipantAccessDenied, got %v", err)
	}
}

func TestFinalizeRoundCommitsReservationsAndCreatesNextRound(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	p1, err := service.JoinRoomWithSeat(ctx, 100, 1, 1)
	if err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	p2, err := service.JoinRoomWithSeat(ctx, 200, 1, 2)
	if err != nil {
		t.Fatalf("second join failed: %v", err)
	}
	if err := scope.UpdateRoundStatus(ctx, 1, "active"); err != nil {
		t.Fatalf("UpdateRoundStatus failed: %v", err)
	}

	if err := service.FinalizeRound(ctx, 1, map[int64]int64{p1: 240, p2: 160}); err != nil {
		t.Fatalf("FinalizeRound failed: %v", err)
	}

	if scope.balances[100] != 1140 {
		t.Fatalf("unexpected winner balance for user 100: got %d want 1140", scope.balances[100])
	}
	if scope.balances[200] != 1060 {
		t.Fatalf("unexpected winner balance for user 200: got %d want 1060", scope.balances[200])
	}
	if scope.rounds[1].ArchivedAt == nil {
		t.Fatal("expected original round to be archived")
	}
	if scope.roomInfo.CurrentRoundID == nil || *scope.roomInfo.CurrentRoundID == 1 {
		t.Fatal("expected next round to be created")
	}
}
