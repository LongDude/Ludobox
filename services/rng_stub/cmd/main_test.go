package main

import "testing"

func TestPickWinningPositionsDeterministicSinglePositiveWeight(t *testing.T) {
	result, err := pickWinningPositions([]float64{0, 0, 1, 0}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected exactly one position, got %d", len(result))
	}
	if result[0] != 3 {
		t.Fatalf("expected position 3, got %d", result[0])
	}
}

func TestPickWinningPositionsNoDuplicates(t *testing.T) {
	result, err := pickWinningPositions([]float64{1, 1, 1, 1}, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 winners, got %d", len(result))
	}

	seen := make(map[int]struct{}, len(result))
	for _, position := range result {
		if position < 1 || position > 4 {
			t.Fatalf("position out of range: %d", position)
		}
		if _, exists := seen[position]; exists {
			t.Fatalf("duplicate position found: %d", position)
		}
		seen[position] = struct{}{}
	}
}

func TestPickWinningPositionsReturnsErrorOnInvalidCount(t *testing.T) {
	_, err := pickWinningPositions([]float64{1, 1}, 3)
	if err == nil {
		t.Fatal("expected error when winners_count > probabilities length")
	}
}

func TestPickWinningPositionsReturnsErrorOnTooManyNonZeroRequired(t *testing.T) {
	_, err := pickWinningPositions([]float64{1, 0, 0}, 2)
	if err == nil {
		t.Fatal("expected error when winners_count > non-zero probabilities count")
	}
}

func TestPickWinningPositionsZeroWinners(t *testing.T) {
	result, err := pickWinningPositions([]float64{0.2, 0.8}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d", len(result))
	}
}
