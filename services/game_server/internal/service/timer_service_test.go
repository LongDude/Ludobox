package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"game_server/internal/domain"
)

func TestStartTimerKeepsRoundWaitingUntilConfiguredCountdownExpires(t *testing.T) {
	scope := newMockTransactionScope()
	scope.participants[1] = &domain.RoundParticipant{
		RoundParticipantID: 1,
		UserID:             100,
		RoundsID:           1,
		NumberInRoom:       1,
	}

	repo := &mockRoomRepository{scope: scope}
	events := NewEventsService(nil)
	timer := NewTimerService(repo, events, nil)

	var startedCalls atomic.Int32
	var finalizedCalls atomic.Int32
	timer.SetGameStartCallback(func(ctx context.Context, roundID int64) error {
		startedCalls.Add(1)
		return nil
	})
	timer.SetGameFinalizeCallback(func(ctx context.Context, roundID int64) error {
		finalizedCalls.Add(1)
		return nil
	})

	timer.StartTimer(context.Background(), 1, 1, 1, 1)
	defer timer.StopTimer(1)

	time.Sleep(500 * time.Millisecond)
	if startedCalls.Load() != 0 {
		t.Fatalf("round started too early: got %d calls", startedCalls.Load())
	}

	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if startedCalls.Load() == 1 && finalizedCalls.Load() == 1 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	if startedCalls.Load() == 1 && finalizedCalls.Load() == 1 {
		return
	}

	t.Fatalf("unexpected callback counts: started=%d finalized=%d", startedCalls.Load(), finalizedCalls.Load())
}
