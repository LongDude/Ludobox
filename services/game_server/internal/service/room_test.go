package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"game_server/internal/domain"
	"game_server/internal/repository"
)

// MockTransactionScope для тестирования
type MockTransactionScope struct {
	balances     map[int64]int64
	participants map[int64]*ParticipantData
	reservations map[int64][]*ReservationData
	shouldFail   map[string]bool
}

type ParticipantData struct {
	UserID       int64
	RoundID      int64
	Boost        int
	WinningMoney int64
	NumberInRoom int
	ExitRoomAt   *time.Time
}

type ReservationData struct {
	ID            int64
	ParticipantID int64
	Type          string
	Amount        int64
	Status        string
	ExpiresAt     time.Time
}

func NewMockTransactionScope() *MockTransactionScope {
	return &MockTransactionScope{
		balances:     make(map[int64]int64),
		participants: make(map[int64]*ParticipantData),
		reservations: make(map[int64][]*ReservationData),
		shouldFail:   make(map[string]bool),
	}
}

// Реализуем методы TransactionScope

func (m *MockTransactionScope) GetBalanceLocked(ctx context.Context, userID int64) (int64, error) {
	if m.shouldFail["GetBalanceLocked"] {
		return 0, errors.New("user not found")
	}
	return m.balances[userID], nil
}

func (m *MockTransactionScope) UpdateBalance(ctx context.Context, userID int64, delta int64) error {
	if m.shouldFail["UpdateBalance"] {
		return errors.New("update failed")
	}
	m.balances[userID] += delta
	return nil
}

func (m *MockTransactionScope) CreateParticipant(ctx context.Context, userID, roundID int64, numberInRoom int) (int64, error) {
	if m.shouldFail["CreateParticipant"] {
		return 0, errors.New("create failed")
	}
	pID := int64(len(m.participants) + 1)
	m.participants[pID] = &ParticipantData{
		UserID:       userID,
		RoundID:      roundID,
		Boost:        0,
		WinningMoney: 0,
		NumberInRoom: numberInRoom,
	}
	return pID, nil
}

func (m *MockTransactionScope) GetParticipantUserID(ctx context.Context, participantID int64) (int64, error) {
	if p, ok := m.participants[participantID]; ok {
		return p.UserID, nil
	}
	return 0, errors.New("participant not found")
}

func (m *MockTransactionScope) UpdateParticipantBoost(ctx context.Context, participantID int64, boost int) error {
	if p, ok := m.participants[participantID]; ok {
		p.Boost = boost
		return nil
	}
	return errors.New("participant not found")
}

func (m *MockTransactionScope) MarkParticipantExited(ctx context.Context, participantID int64) error {
	if p, ok := m.participants[participantID]; ok {
		now := time.Now()
		p.ExitRoomAt = &now
		return nil
	}
	return errors.New("participant not found")
}

func (m *MockTransactionScope) UpdateWinningMoney(ctx context.Context, participantID int64, amount int64) error {
	if p, ok := m.participants[participantID]; ok {
		p.WinningMoney = amount
		return nil
	}
	return errors.New("participant not found")
}

func (m *MockTransactionScope) ReserveEntry(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	if m.shouldFail["ReserveEntry"] {
		return 0, errors.New("reserve failed")
	}
	rID := int64(len(m.reservations) + 1)
	m.reservations[participantID] = append(m.reservations[participantID], &ReservationData{
		ID:            rID,
		ParticipantID: participantID,
		Type:          "entry_fee",
		Amount:        amount,
		Status:        "active",
		ExpiresAt:     expiresAt,
	})
	return rID, nil
}

func (m *MockTransactionScope) ReserveBoost(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	if m.shouldFail["ReserveBoost"] {
		return 0, errors.New("reserve failed")
	}
	rID := int64(len(m.reservations) + 1)
	m.reservations[participantID] = append(m.reservations[participantID], &ReservationData{
		ID:            rID,
		ParticipantID: participantID,
		Type:          "boost",
		Amount:        amount,
		Status:        "active",
		ExpiresAt:     expiresAt,
	})
	return rID, nil
}

func (m *MockTransactionScope) ReleaseAllReservations(ctx context.Context, participantID int64) (int64, error) {
	var sum int64
	if reservations, ok := m.reservations[participantID]; ok {
		for _, r := range reservations {
			if r.Status == "active" {
				sum += r.Amount
				r.Status = "released"
			}
		}
	}
	return sum, nil
}

func (m *MockTransactionScope) ReleaseBoostReservations(ctx context.Context, participantID int64) (int64, error) {
	var sum int64
	hasActive := false
	if reservations, ok := m.reservations[participantID]; ok {
		for _, r := range reservations {
			if r.Type == "boost" && r.Status == "active" {
				sum += r.Amount
				r.Status = "released"
				hasActive = true
			}
		}
	}
	if !hasActive {
		return 0, repository.ErrActiveReservationNotFound
	}
	return sum, nil
}

func (m *MockTransactionScope) CommitReservations(ctx context.Context, participantID int64) (int64, error) {
	var sum int64
	hasActive := false
	if reservations, ok := m.reservations[participantID]; ok {
		for _, r := range reservations {
			if r.Status == "active" {
				sum += r.Amount
				r.Status = "committed"
				hasActive = true
			}
		}
	}
	if !hasActive {
		return 0, repository.ErrActiveReservationNotFound
	}
	return sum, nil
}

func (m *MockTransactionScope) ArchiveRound(ctx context.Context, roundID int64) error {
	return nil
}

// MockRoomRepository
type MockRoomRepository struct {
	ts repository.TransactionScope
}

func (m *MockRoomRepository) InTransaction(ctx context.Context, fn func(ts repository.TransactionScope) error) error {
	return fn(m.ts)
}

func (m *MockRoomRepository) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	return nil, nil
}

func (m *MockRoomRepository) GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	return nil, nil
}

// Тесты

func TestJoinRoom_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockTS := NewMockTransactionScope()
	mockTS.balances[100] = 1000 // user 100 имеет 1000 баланса

	mockRepo := &MockRoomRepository{ts: mockTS}
	svc := NewRoomService(mockRepo, nil)

	// Act
	participantID, err := svc.JoinRoom(ctx, 100, 1, 100, 10, 5, time.Now().Add(1*time.Hour))

	// Assert
	if err != nil {
		t.Fatalf("JoinRoom failed: %v", err)
	}
	if participantID == 0 {
		t.Fatal("participantID should not be 0")
	}
	if mockTS.balances[100] != 900 { // 1000 - 100 = 900
		t.Errorf("balance not updated: got %d, want 900", mockTS.balances[100])
	}
	if len(mockTS.participants) != 1 {
		t.Errorf("participant not created: got %d participants", len(mockTS.participants))
	}
	if len(mockTS.reservations[participantID]) != 1 {
		t.Errorf("reservation not created: got %d reservations", len(mockTS.reservations[participantID]))
	}
}

func TestJoinRoom_InsufficientBalance(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockTS := NewMockTransactionScope()
	mockTS.balances[100] = 50 // user 100 имеет только 50

	mockRepo := &MockRoomRepository{ts: mockTS}
	svc := NewRoomService(mockRepo, nil)

	// Act
	_, err := svc.JoinRoom(ctx, 100, 1, 100, 10, 5, time.Now().Add(1*time.Hour))

	// Assert
	if err == nil {
		t.Fatal("JoinRoom should fail with insufficient balance")
	}
	if !errors.Is(err, repository.ErrInsufficientBalance) {
		t.Errorf("wrong error: got %v, want ErrInsufficientBalance", err)
	}
}

func TestJoinRoom_RoomIsFull(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockTS := NewMockTransactionScope()
	mockTS.balances[100] = 1000

	mockRepo := &MockRoomRepository{ts: mockTS}
	svc := NewRoomService(mockRepo, nil)

	// Act - room with capacity 5, already has 5 players
	_, err := svc.JoinRoom(ctx, 100, 1, 100, 5, 5, time.Now().Add(1*time.Hour))

	// Assert
	if err == nil {
		t.Fatal("JoinRoom should fail when room is full")
	}
	if !errors.Is(err, repository.ErrRoomIsFull) {
		t.Errorf("wrong error: got %v, want ErrRoomIsFull", err)
	}
}

func TestPurchaseBoost_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockTS := NewMockTransactionScope()
	mockTS.balances[100] = 1000
	mockTS.participants[1] = &ParticipantData{UserID: 100, RoundID: 1, Boost: 0}

	mockRepo := &MockRoomRepository{ts: mockTS}
	svc := NewRoomService(mockRepo, nil)

	// Act
	err := svc.PurchaseBoost(ctx, 1, 100, 50, 200, time.Now().Add(1*time.Hour))

	// Assert
	if err != nil {
		t.Fatalf("PurchaseBoost failed: %v", err)
	}
	if mockTS.balances[100] != 800 { // 1000 - 200 = 800
		t.Errorf("balance not updated: got %d, want 800", mockTS.balances[100])
	}
	if mockTS.participants[1].Boost != 50 {
		t.Errorf("boost not updated: got %d, want 50", mockTS.participants[1].Boost)
	}
}

func TestLeaveRoom_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockTS := NewMockTransactionScope()
	mockTS.balances[100] = 900
	mockTS.participants[1] = &ParticipantData{UserID: 100, RoundID: 1}
	mockTS.reservations[1] = []*ReservationData{
		{ID: 1, ParticipantID: 1, Type: "entry_fee", Amount: 100, Status: "active"},
	}

	mockRepo := &MockRoomRepository{ts: mockTS}
	svc := NewRoomService(mockRepo, nil)

	// Act
	err := svc.LeaveRoom(ctx, 1, 100)

	// Assert
	if err != nil {
		t.Fatalf("LeaveRoom failed: %v", err)
	}
	if mockTS.balances[100] != 1000 { // 900 + 100 (refund) = 1000
		t.Errorf("balance not refunded: got %d, want 1000", mockTS.balances[100])
	}
	if mockTS.participants[1].ExitRoomAt == nil {
		t.Error("participant exit_room_at not set")
	}
}

func TestFinalizeRound_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockTS := NewMockTransactionScope()
	mockTS.balances[100] = 0
	mockTS.participants[1] = &ParticipantData{UserID: 100, RoundID: 1, WinningMoney: 0}
	mockTS.reservations[1] = []*ReservationData{
		{ID: 1, ParticipantID: 1, Type: "entry_fee", Amount: 100, Status: "active"},
	}

	mockRepo := &MockRoomRepository{ts: mockTS}
	svc := NewRoomService(mockRepo, nil)

	// Act
	winners := map[int64]int64{1: 500} // participant 1 wins 500
	err := svc.FinalizeRound(ctx, 1, winners)

	// Assert
	if err != nil {
		t.Fatalf("FinalizeRound failed: %v", err)
	}
	if mockTS.balances[100] != 500 { // 0 + 500 (win) = 500
		t.Errorf("balance not updated: got %d, want 500", mockTS.balances[100])
	}
	if mockTS.participants[1].WinningMoney != 500 {
		t.Errorf("winning money not updated: got %d, want 500", mockTS.participants[1].WinningMoney)
	}
}
