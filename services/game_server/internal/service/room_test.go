package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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
	roomEvents        []domain.RoomEvent
	reservations      map[int64][]*reservationData
	balances          map[int64]int64
	ratings           map[int64]int64
	ratingRewards     []domain.UserRatingReward
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
				RoundTime:           60,
				NextRoundDelay:      0,
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
		ratings:           map[int64]int64{100: 0, 200: 0},
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

func (m *mockTransactionScope) ApplyUserRatingReward(ctx context.Context, reward domain.UserRatingReward) error {
	m.ratings[reward.UserID] += reward.Delta
	m.ratingRewards = append(m.ratingRewards, reward)
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

func (m *mockTransactionScope) SetRoomCurrentPlayers(ctx context.Context, roomID int64, currentPlayers int) error {
	if roomID != m.roomInfo.Room.RoomID {
		return repository.ErrRoomNotFound
	}
	m.roomInfo.Room.CurrentPlayers = currentPlayers
	return nil
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

func (m *mockRoomRepository) CreateRoomEvent(ctx context.Context, roomID int64, roundID *int64, eventType string, eventData json.RawMessage) error {
	m.scope.roomEvents = append(m.scope.roomEvents, domain.RoomEvent{
		RoomEventID: int64(len(m.scope.roomEvents) + 1),
		RoomID:      roomID,
		RoundID:     roundID,
		EventType:   eventType,
		EventData:   eventData,
		CreatedAt:   time.Now(),
	})
	return nil
}

func (m *mockRoomRepository) ListRecentRoomEvents(ctx context.Context, roomID int64, limit int) ([]domain.RoomEvent, error) {
	events := make([]domain.RoomEvent, 0, limit)
	for _, event := range m.scope.roomEvents {
		if event.RoomID == roomID {
			events = append(events, event)
		}
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	for left, right := 0, len(events)-1; left < right; left, right = left+1, right-1 {
		events[left], events[right] = events[right], events[left]
	}
	return events, nil
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
	if scope.roomInfo.Room.CurrentPlayers != 1 {
		t.Fatalf("unexpected current players: got %d want 1", scope.roomInfo.Room.CurrentPlayers)
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

	if _, err := service.JoinRoomWithSeat(ctx, 100, 1, 3); !errors.Is(err, repository.ErrMaxSeatsExceeded) {
		t.Fatalf("expected ErrMaxSeatsExceeded, got %v", err)
	}
}

func TestJoinRoomAllocatesAnotherRandomSeatForSameUser(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	participantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 2)
	if err != nil {
		t.Fatalf("JoinRoomWithSeat failed: %v", err)
	}

	reusedParticipantID, err := service.JoinRoom(ctx, 100, 1)
	if err != nil {
		t.Fatalf("JoinRoom failed: %v", err)
	}

	if reusedParticipantID == participantID {
		t.Fatalf("expected a new participant, got reused id %d", reusedParticipantID)
	}
	if len(scope.participants) != 2 {
		t.Fatalf("unexpected participants count: got %d want 2", len(scope.participants))
	}
	if scope.balances[100] != 800 {
		t.Fatalf("unexpected balance after second random join: got %d want 800", scope.balances[100])
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

func TestPurchaseBoostAllowsOnlyOneBoostPerUser(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	firstParticipantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 1)
	if err != nil {
		t.Fatalf("first JoinRoomWithSeat failed: %v", err)
	}
	secondParticipantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 2)
	if err != nil {
		t.Fatalf("second JoinRoomWithSeat failed: %v", err)
	}

	if err := service.PurchaseBoost(ctx, firstParticipantID, 100, 25, 50); err != nil {
		t.Fatalf("PurchaseBoost failed: %v", err)
	}
	if scope.participants[firstParticipantID].Boost != 25 {
		t.Fatalf("unexpected boost: got %d want 25", scope.participants[firstParticipantID].Boost)
	}
	if scope.balances[100] != 750 {
		t.Fatalf("unexpected balance after boost: got %d want 750", scope.balances[100])
	}

	err = service.PurchaseBoost(ctx, secondParticipantID, 100, 25, 50)
	if !errors.Is(err, repository.ErrBoostAlreadyPurchased) {
		t.Fatalf("expected ErrBoostAlreadyPurchased, got %v", err)
	}
	if scope.participants[secondParticipantID].Boost != 0 {
		t.Fatalf("unexpected second participant boost: got %d want 0", scope.participants[secondParticipantID].Boost)
	}
	if scope.balances[100] != 750 {
		t.Fatalf("unexpected balance after rejected boost: got %d want 750", scope.balances[100])
	}
}

func TestPurchaseBoostRejectedDuringActiveRound(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	participantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 1)
	if err != nil {
		t.Fatalf("JoinRoomWithSeat failed: %v", err)
	}
	if err := scope.UpdateRoundStatus(ctx, 1, "active"); err != nil {
		t.Fatalf("UpdateRoundStatus failed: %v", err)
	}

	if err := service.PurchaseBoost(ctx, participantID, 100, 25, 50); !errors.Is(err, repository.ErrGameAlreadyStarted) {
		t.Fatalf("expected ErrGameAlreadyStarted, got %v", err)
	}
	if scope.participants[participantID].Boost != 0 {
		t.Fatalf("unexpected boost: got %d want 0", scope.participants[participantID].Boost)
	}
	if scope.balances[100] != 900 {
		t.Fatalf("unexpected balance: got %d want 900", scope.balances[100])
	}
}

func TestLeaveRoomByUserReleasesAllUserSeats(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	firstParticipantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 1)
	if err != nil {
		t.Fatalf("first JoinRoomWithSeat failed: %v", err)
	}
	secondParticipantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 2)
	if err != nil {
		t.Fatalf("second JoinRoomWithSeat failed: %v", err)
	}
	otherParticipantID, err := service.JoinRoomWithSeat(ctx, 200, 1, 3)
	if err != nil {
		t.Fatalf("other JoinRoomWithSeat failed: %v", err)
	}
	if err := service.PurchaseBoost(ctx, firstParticipantID, 100, 25, 50); err != nil {
		t.Fatalf("PurchaseBoost failed: %v", err)
	}

	refund, err := service.LeaveRoomByUser(ctx, 100, 1)
	if err != nil {
		t.Fatalf("LeaveRoomByUser failed: %v", err)
	}

	if refund != 250 {
		t.Fatalf("unexpected refund: got %d want 250", refund)
	}
	if scope.balances[100] != 1000 {
		t.Fatalf("unexpected user balance: got %d want 1000", scope.balances[100])
	}
	if scope.participants[firstParticipantID].ExitRoomAt == nil {
		t.Fatal("expected first participant to exit")
	}
	if scope.participants[secondParticipantID].ExitRoomAt == nil {
		t.Fatal("expected second participant to exit")
	}
	if scope.participants[otherParticipantID].ExitRoomAt != nil {
		t.Fatal("expected other user's participant to remain active")
	}
	activeCount, err := scope.GetActiveParticipantsCount(ctx, 1)
	if err != nil {
		t.Fatalf("GetActiveParticipantsCount failed: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("unexpected active count: got %d want 1", activeCount)
	}
	if scope.roomInfo.Room.CurrentPlayers != 1 {
		t.Fatalf("unexpected current players: got %d want 1", scope.roomInfo.Room.CurrentPlayers)
	}
}

func TestLeaveRoomByUserIgnoresFinishedRound(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	if _, err := service.JoinRoomWithSeat(ctx, 100, 1, 1); err != nil {
		t.Fatalf("JoinRoomWithSeat failed: %v", err)
	}
	if err := scope.UpdateRoundStatus(ctx, 1, "finished"); err != nil {
		t.Fatalf("UpdateRoundStatus failed: %v", err)
	}

	refund, err := service.LeaveRoomByUser(ctx, 100, 1)
	if err != nil {
		t.Fatalf("LeaveRoomByUser failed: %v", err)
	}
	if refund != 0 {
		t.Fatalf("unexpected refund: got %d want 0", refund)
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
	expectedReward := calculateRatingReward(scope.roomInfo.Config)
	if scope.ratings[100] != expectedReward {
		t.Fatalf("unexpected winner rating for user 100: got %d want %d", scope.ratings[100], expectedReward)
	}
	if scope.ratings[200] != expectedReward {
		t.Fatalf("unexpected winner rating for user 200: got %d want %d", scope.ratings[200], expectedReward)
	}
	if scope.rounds[1].ArchivedAt == nil {
		t.Fatal("expected original round to be archived")
	}
	if scope.roomInfo.CurrentRoundID == nil || *scope.roomInfo.CurrentRoundID == 1 {
		t.Fatal("expected next round to be created")
	}
	if scope.roomInfo.Room.CurrentPlayers != 0 {
		t.Fatalf("unexpected current players after finalize: got %d want 0", scope.roomInfo.Room.CurrentPlayers)
	}
}

func TestRecoverServerStateCancelsInterruptedActiveRound(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}
	service := NewRoomService(repo, nil, 1, nil, "")

	firstParticipantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 1)
	if err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	secondParticipantID, err := service.JoinRoomWithSeat(ctx, 200, 1, 2)
	if err != nil {
		t.Fatalf("second join failed: %v", err)
	}
	if err := scope.UpdateRoundStatus(ctx, 1, "active"); err != nil {
		t.Fatalf("UpdateRoundStatus failed: %v", err)
	}

	if err := service.RecoverServerState(ctx); err != nil {
		t.Fatalf("RecoverServerState failed: %v", err)
	}

	if scope.rounds[1].Status != "cancelled" {
		t.Fatalf("unexpected round status: got %s want cancelled", scope.rounds[1].Status)
	}
	if scope.rounds[1].ArchivedAt == nil {
		t.Fatal("expected interrupted round to be archived")
	}
	if scope.participants[firstParticipantID].ExitRoomAt == nil || scope.participants[secondParticipantID].ExitRoomAt == nil {
		t.Fatal("expected all participants to be marked exited")
	}
	if scope.balances[100] != 1000 || scope.balances[200] != 1000 {
		t.Fatalf("unexpected balances after recovery: got %d and %d want 1000 and 1000", scope.balances[100], scope.balances[200])
	}
	if scope.roomInfo.Room.CurrentPlayers != 0 {
		t.Fatalf("unexpected current players after recovery: got %d want 0", scope.roomInfo.Room.CurrentPlayers)
	}
}

func TestBuildPayoutsUsesActualParticipantsAndCommission(t *testing.T) {
	config := &domain.RoomConfig{
		Capacity:            2,
		RegistrationPrice:   100,
		Commission:          10,
		WinningDistribution: []int{60, 40},
		RoundTime:           60,
	}

	payouts := buildPayouts(config, 2, 2)
	if len(payouts) != 2 {
		t.Fatalf("unexpected payouts length: got %d want 2", len(payouts))
	}
	if payouts[0] != 108 {
		t.Fatalf("unexpected first payout: got %d want 108", payouts[0])
	}
	if payouts[1] != 72 {
		t.Fatalf("unexpected second payout: got %d want 72", payouts[1])
	}
}

func TestBuildGrossPayoutsKeepsPreCommissionDistribution(t *testing.T) {
	config := &domain.RoomConfig{
		Capacity:            2,
		RegistrationPrice:   100,
		Commission:          10,
		WinningDistribution: []int{60, 40},
	}

	payouts := buildGrossPayouts(config, 2)
	if len(payouts) != 2 {
		t.Fatalf("unexpected payouts length: got %d want 2", len(payouts))
	}
	if payouts[0] != 120 {
		t.Fatalf("unexpected first gross payout: got %d want 120", payouts[0])
	}
	if payouts[1] != 80 {
		t.Fatalf("unexpected second gross payout: got %d want 80", payouts[1])
	}
}

func TestRequestWinningPositionsMapsWinnersToActualSeatNumbers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/winnings/distribute" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}

		var payload rngDistributeRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(payload.Probabilities) != 4 {
			t.Fatalf("unexpected probabilities length: got %d want 4", len(payload.Probabilities))
		}
		expectedProbabilities := []float64{0.25, 0.25, 0.25, 0.3125}
		for idx, probability := range payload.Probabilities {
			if probability != expectedProbabilities[idx] {
				t.Fatalf("unexpected probability at %d: got %v want %v", idx, probability, expectedProbabilities[idx])
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(rngDistributeResponse{WinningPositions: []int{2, 4}}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	service := &RoomService{
		rngDistributeURL: server.URL + "/winnings/distribute",
		httpClient:       server.Client(),
	}

	positions, err := service.requestWinningPositions(context.Background(), &domain.RoomConfig{
		Capacity:      4,
		NumberWinners: 2,
		BoostPower:    25,
	}, appendBotParticipants(1, 4, []domain.RoundParticipant{
		{RoundParticipantID: 10, NumberInRoom: 4, Boost: 25},
		{RoundParticipantID: 11, NumberInRoom: 2},
	}))
	if err != nil {
		t.Fatalf("request winning positions: %v", err)
	}

	if len(positions) != 2 {
		t.Fatalf("unexpected positions length: got %d want 2", len(positions))
	}
	if positions[0] != 2 || positions[1] != 4 {
		t.Fatalf("unexpected mapped positions: got %v want [2 4]", positions)
	}
}

func TestFinalizeGameRoundIncludesBotWinnersWithoutCreditingBots(t *testing.T) {
	ctx := context.Background()
	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload rngDistributeRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if err := json.NewEncoder(writer).Encode(rngDistributeResponse{WinningPositions: []int{1, 2}}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	service := NewRoomService(repo, nil, 1, nil, server.URL)
	service.httpClient = server.Client()

	firstParticipantID, err := service.JoinRoomWithSeat(ctx, 100, 1, 2)
	if err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	secondParticipantID, err := service.JoinRoomWithSeat(ctx, 200, 1, 4)
	if err != nil {
		t.Fatalf("second join failed: %v", err)
	}
	if err := scope.UpdateRoundStatus(ctx, 1, "active"); err != nil {
		t.Fatalf("UpdateRoundStatus failed: %v", err)
	}

	winners, err := service.FinalizeGameRound(ctx, 1)
	if err != nil {
		t.Fatalf("FinalizeGameRound failed: %v", err)
	}

	if len(winners) != 2 {
		t.Fatalf("unexpected winners count: got %d want 2", len(winners))
	}
	if !winners[0].IsBot || winners[0].NumberInRoom != 1 {
		t.Fatalf("expected first winner to be bot in seat 1, got %+v", winners[0])
	}
	if winners[1].IsBot || winners[1].RoundParticipantID != firstParticipantID || winners[1].WinningMoney != 160 {
		t.Fatalf("expected second winner to be participant %d with 160, got %+v", firstParticipantID, winners[1])
	}
	if scope.participants[firstParticipantID].WinningMoney != 160 {
		t.Fatalf("unexpected winning money for first participant: got %d want 160", scope.participants[firstParticipantID].WinningMoney)
	}
	if scope.participants[secondParticipantID].WinningMoney != 0 {
		t.Fatalf("unexpected winning money for second participant: got %d want 0", scope.participants[secondParticipantID].WinningMoney)
	}
	if scope.balances[100] != 1060 {
		t.Fatalf("unexpected balance for user 100: got %d want 1060", scope.balances[100])
	}
	if scope.balances[200] != 900 {
		t.Fatalf("unexpected balance for user 200: got %d want 900", scope.balances[200])
	}
	expectedReward := calculateRatingReward(scope.roomInfo.Config)
	if scope.ratings[100] != expectedReward {
		t.Fatalf("unexpected rating for user 100: got %d want %d", scope.ratings[100], expectedReward)
	}
	if scope.ratings[200] != 0 {
		t.Fatalf("unexpected rating for user 200: got %d want 0", scope.ratings[200])
	}
}

func TestCalculateRatingRewardRewardsRiskierConfigsMore(t *testing.T) {
	safer := calculateRatingReward(&domain.RoomConfig{
		Capacity:          4,
		RegistrationPrice: 50,
		NumberWinners:     2,
		Commission:        5,
	})
	riskier := calculateRatingReward(&domain.RoomConfig{
		Capacity:          10,
		RegistrationPrice: 5000,
		NumberWinners:     1,
		Commission:        20,
		IsBoost:           true,
		BoostPower:        60,
	})

	if riskier <= safer {
		t.Fatalf("expected riskier config to reward more rating: safe=%d risky=%d", safer, riskier)
	}
}

func TestFinalizeGameRoundDoesNotCancelTimerContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scope := newMockTransactionScope()
	repo := &mockRoomRepository{scope: scope}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := json.NewEncoder(writer).Encode(rngDistributeResponse{WinningPositions: []int{1, 2}}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	events := NewEventsService(repo, nil)
	timer := NewTimerService(repo, events, nil)
	var timerCancelled atomic.Bool
	timer.mu.Lock()
	timer.timers[1] = &timerState{
		cancel: func() {
			timerCancelled.Store(true)
			cancel()
		},
		status:   "active",
		deadline: time.Now().Add(time.Minute),
	}
	timer.mu.Unlock()
	defer timer.StopTimer(1)

	service := NewRoomService(repo, nil, 1, nil, server.URL)
	service.httpClient = server.Client()
	service.SetTimerService(timer)

	if _, err := service.JoinRoomWithSeat(ctx, 100, 1, 1); err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if _, err := service.JoinRoomWithSeat(ctx, 200, 1, 2); err != nil {
		t.Fatalf("second join failed: %v", err)
	}
	if err := scope.UpdateRoundStatus(ctx, 1, "active"); err != nil {
		t.Fatalf("UpdateRoundStatus failed: %v", err)
	}

	if _, err := service.FinalizeGameRound(ctx, 1); err != nil {
		t.Fatalf("FinalizeGameRound failed: %v", err)
	}

	if timerCancelled.Load() {
		t.Fatal("FinalizeGameRound must not stop the timer before round_finalized can be published")
	}
	if ctx.Err() != nil {
		t.Fatalf("FinalizeGameRound cancelled caller context: %v", ctx.Err())
	}
	if scope.rounds[1].Status != "finished" {
		t.Fatalf("expected finished status, got %s", scope.rounds[1].Status)
	}
}
