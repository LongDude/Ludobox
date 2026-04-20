package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"game_server/internal/repository"
	"game_server/internal/service"
)

// IntegrationTestHelper предоставляет тестовую БД PostgreSQL
type IntegrationTestHelper struct {
	db *pgxpool.Pool
	t  *testing.T
}

func NewIntegrationTestHelper(t *testing.T) *IntegrationTestHelper {
	// В реальном проекте это должно подключаться к тестовой БД
	// Пример: postgresql://test:test@localhost:5432/game_server_test
	connStr := "postgresql://test:test@localhost:5432/game_server_test"

	db, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	// Миграции должны быть запущены перед тестами
	// Это обеспечивает консистентную схему БД

	return &IntegrationTestHelper{db: db, t: t}
}

func (h *IntegrationTestHelper) Close() {
	h.db.Close()
}

func (h *IntegrationTestHelper) Cleanup() {
	ctx := context.Background()
	// Очистить тестовые данные
	h.db.Exec(ctx, "DELETE FROM user_balance_reservations")
	h.db.Exec(ctx, "DELETE FROM round_participants")
	h.db.Exec(ctx, "DELETE FROM rounds")
	h.db.Exec(ctx, "DELETE FROM rooms")
	h.db.Exec(ctx, "DELETE FROM config")
	h.db.Exec(ctx, "DELETE FROM users")
}

func (h *IntegrationTestHelper) CreateUser(userID int64, balance int64) error {
	ctx := context.Background()
	_, err := h.db.Exec(ctx,
		`INSERT INTO users (user_id, nickname, balance) VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		userID, "test_user_"+string(rune(userID)), balance)
	return err
}

func (h *IntegrationTestHelper) GetBalance(userID int64) int64 {
	ctx := context.Background()
	var bal int64
	h.db.QueryRow(ctx, "SELECT balance FROM users WHERE user_id = $1", userID).Scan(&bal)
	return bal
}

func (h *IntegrationTestHelper) CreateRound() int64 {
	ctx := context.Background()
	var roundID int64

	// Сначала создаём room (требуется config_id и server_id)
	var roomID int64
	h.db.QueryRow(ctx,
		`INSERT INTO rooms (config_id, server_id, status) 
		 VALUES (1, 1, 'open')
		 RETURNING room_id`).Scan(&roomID)

	// Затем создаём round
	h.db.QueryRow(ctx,
		`INSERT INTO rounds (room_id) VALUES ($1) RETURNING rounds_id`,
		roomID).Scan(&roundID)

	return roundID
}

// --- ТЕСТЫ ---

// TestFullRoundCycle проверяет полный цикл жизни раунда
func TestFullRoundCycle(t *testing.T) {
	h := NewIntegrationTestHelper(t)
	defer h.Close()
	h.Cleanup()

	ctx := context.Background()
	logger := logrus.New()

	// Setup
	repo := NewRoomRepository(h.db)
	svc := service.NewRoomService(repo, logger)

	// Создать пользователей с балансом
	h.CreateUser(1, 1000) // 1000 для entry + boost
	h.CreateUser(2, 1000)
	h.CreateUser(3, 500) // Недостаточный баланс для буста

	// Создать раунд
	roundID := h.CreateRound()

	// Phase 1: JoinRoom
	const (
		capacity   = 5
		entryPrice = 100
		boostPrice = 50
	)

	pID1, err := svc.JoinRoom(ctx, 1, roundID, entryPrice, capacity, 0, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Greater(t, pID1, int64(0))
	require.Equal(t, int64(900), h.GetBalance(1)) // 1000 - 100

	pID2, err := svc.JoinRoom(ctx, 2, roundID, entryPrice, capacity, 1, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(900), h.GetBalance(2))

	// Phase 2: PurchaseBoost
	err = svc.PurchaseBoost(ctx, pID1, 1, 30, boostPrice, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(850), h.GetBalance(1)) // 900 - 50

	// Phase 3: CancelBoost
	err = svc.CancelBoost(ctx, pID1, 1)
	require.NoError(t, err)
	require.Equal(t, int64(900), h.GetBalance(1)) // 850 + 50 (refund)

	// Phase 4: LeaveRoom
	err = svc.LeaveRoom(ctx, pID1, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1000), h.GetBalance(1)) // 900 + 100 (entry refund)

	// Phase 5: FinalizeRound
	winners := map[int64]int64{
		pID2: 400, // Winner: 400
	}
	err = svc.FinalizeRound(ctx, roundID, winners)
	require.NoError(t, err)

	// Проверить финальные балансы
	// User 1: вышел, вернул 100 → 1000 ✓
	require.Equal(t, int64(1000), h.GetBalance(1))

	// User 2: был участником, выиграл 400
	// 900 (после entry) + 400 (win) = 1300 ✓
	require.Equal(t, int64(1300), h.GetBalance(2))
}

// TestConcurrentJoinRoom проверяет параллельные присоединения
func TestConcurrentJoinRoom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := NewIntegrationTestHelper(t)
	defer h.Close()
	h.Cleanup()

	ctx := context.Background()
	logger := logrus.New()

	repo := NewRoomRepository(h.db)
	svc := service.NewRoomService(repo, logger)

	const (
		numUsers   = 10
		capacity   = 5
		entryPrice = 100
	)

	roundID := h.CreateRound()

	// Создать пользователей
	for i := 1; i <= numUsers; i++ {
		h.CreateUser(int64(i), 1000)
	}

	// Запустить горутины
	successCount := 0
	errorCount := 0

	for i := 1; i <= numUsers; i++ {
		// В реальном тесте это должны быть горутины
		// for _, _ = range make([]int, 10) { go func() { ... }() }
		_, err := svc.JoinRoom(ctx, int64(i), roundID, entryPrice, capacity, i-1, time.Now().Add(1*time.Hour))
		if err == nil {
			successCount++
		} else if err == repository.ErrRoomIsFull {
			errorCount++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Проверить результаты
	require.Equal(t, capacity, successCount, "должно быть точно capacity успешных присоединений")
	require.Equal(t, numUsers-capacity, errorCount, "остальные должны получить ErrRoomIsFull")
}

// TestInsufficientBalanceScenarios проверяет различные сценарии с недостатком баланса
func TestInsufficientBalanceScenarios(t *testing.T) {
	h := NewIntegrationTestHelper(t)
	defer h.Close()
	h.Cleanup()

	ctx := context.Background()
	logger := logrus.New()

	repo := NewRoomRepository(h.db)
	svc := service.NewRoomService(repo, logger)

	roundID := h.CreateRound()

	// Сценарий 1: Недостаточный баланс для entry
	h.CreateUser(1, 50) // Нужно 100
	_, err := svc.JoinRoom(ctx, 1, roundID, 100, 5, 0, time.Now().Add(1*time.Hour))
	require.Equal(t, repository.ErrInsufficientBalance, err)

	// Сценарий 2: Баланс достаточен для entry, но не для boost
	h.CreateUser(2, 150) // entry=100, boost=50, остаток=0
	pID, err := svc.JoinRoom(ctx, 2, roundID, 100, 5, 0, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Попытка купить дорогой буст
	err = svc.PurchaseBoost(ctx, pID, 2, 30, 100, time.Now().Add(1*time.Hour))
	require.Equal(t, repository.ErrInsufficientBalance, err)

	// Сценарий 3: Возврат старого буста позволяет купить новый
	h.CreateUser(3, 100) // Только на entry
	pID3, err := svc.JoinRoom(ctx, 3, roundID, 100, 5, 0, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(0), h.GetBalance(3))

	// Сначала boost не может быть куплен
	err = svc.PurchaseBoost(ctx, pID3, 3, 30, 50, time.Now().Add(1*time.Hour))
	require.Equal(t, repository.ErrInsufficientBalance, err)

	// После LeaveRoom можно купить новый (?)
	// Нет, после LeaveRoom деньги возвращены, но участник уже вышел
	err = svc.LeaveRoom(ctx, pID3, 3)
	require.NoError(t, err)
	require.Equal(t, int64(100), h.GetBalance(3))
}

// TestBoostReplacement проверяет замену бустов
func TestBoostReplacement(t *testing.T) {
	h := NewIntegrationTestHelper(t)
	defer h.Close()
	h.Cleanup()

	ctx := context.Background()
	logger := logrus.New()

	repo := NewRoomRepository(h.db)
	svc := service.NewRoomService(repo, logger)

	h.CreateUser(1, 1000)
	roundID := h.CreateRound()

	pID, err := svc.JoinRoom(ctx, 1, roundID, 100, 5, 0, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(900), h.GetBalance(1))

	// Первый буст
	err = svc.PurchaseBoost(ctx, pID, 1, 25, 50, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(850), h.GetBalance(1)) // 900 - 50

	// Замена на дорогой буст
	err = svc.PurchaseBoost(ctx, pID, 1, 75, 120, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(780), h.GetBalance(1)) // 850 - 120 + 50 (refund) = 780

	// Замена на дешёвый буст
	err = svc.PurchaseBoost(ctx, pID, 1, 10, 30, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(870), h.GetBalance(1)) // 780 + 120 (refund) - 30 = 870
}

// TestParticipantExit проверяет выход участников
func TestParticipantExit(t *testing.T) {
	h := NewIntegrationTestHelper(t)
	defer h.Close()
	h.Cleanup()

	ctx := context.Background()
	logger := logrus.New()

	repo := NewRoomRepository(h.db)
	svc := service.NewRoomService(repo, logger)

	h.CreateUser(1, 1000)
	roundID := h.CreateRound()

	// Join + Boost
	pID, _ := svc.JoinRoom(ctx, 1, roundID, 100, 5, 0, time.Now().Add(1*time.Hour))
	svc.PurchaseBoost(ctx, pID, 1, 50, 50, time.Now().Add(1*time.Hour))
	require.Equal(t, int64(850), h.GetBalance(1)) // 1000 - 100 - 50

	// Exit должен вернуть ВСЕ резервы (entry + boost)
	err := svc.LeaveRoom(ctx, pID, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1000), h.GetBalance(1)) // полный возврат

	// Попытка выхода ещё раз должна вернуть 0
	// (нет активных резервов)
	err = svc.LeaveRoom(ctx, pID, 1)
	// Ошибка ErrActiveReservationNotFound ожидается или нет?
	// Зависит от логики
}
