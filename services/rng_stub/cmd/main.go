package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultMin = int64(0)
	defaultMax = int64(1000)
)

type randomResponse struct {
	Number int64 `json:"number"`
	Min    int64 `json:"min"`
	Max    int64 `json:"max"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type distributeWinningsRequest struct {
	Probabilities []float64 `json:"probabilities"`
	WinnersCount  int       `json:"winners_count"`
}

type distributeWinningsResponse struct {
	WinningPositions []int `json:"winning_positions"`
	WinnersCount     int   `json:"winners_count"`
	PositionsCount   int   `json:"positions_count"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/random", randomHandler)
	mux.HandleFunc("/winnings/distribute", distributeWinningsHandler)

	port := getenv("HTTP_PORT", "8080")
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("rng-stub listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("rng-stub failed: %v", err)
	}
}

func healthHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeJSON(writer, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func randomHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeJSON(writer, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	min, max, err := parseRange(request)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	number, err := randomInt64(min, max)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, errorResponse{Error: "failed to generate random number"})
		return
	}

	writeJSON(writer, http.StatusOK, randomResponse{
		Number: number,
		Min:    min,
		Max:    max,
	})
}

func distributeWinningsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(writer, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()

	var payload distributeWinningsRequest
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "JSON payload must contain a single object"})
		return
	}

	positions, err := pickWinningPositions(payload.Probabilities, payload.WinnersCount)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(writer, http.StatusOK, distributeWinningsResponse{
		WinningPositions: positions,
		WinnersCount:     payload.WinnersCount,
		PositionsCount:   len(payload.Probabilities),
	})
}

func parseRange(request *http.Request) (int64, int64, error) {
	query := request.URL.Query()

	min := defaultMin
	max := defaultMax

	if minRaw := query.Get("min"); minRaw != "" {
		value, err := strconv.ParseInt(minRaw, 10, 64)
		if err != nil {
			return 0, 0, errors.New("invalid min parameter")
		}
		min = value
	}

	if maxRaw := query.Get("max"); maxRaw != "" {
		value, err := strconv.ParseInt(maxRaw, 10, 64)
		if err != nil {
			return 0, 0, errors.New("invalid max parameter")
		}
		max = value
	}

	if min > max {
		return 0, 0, errors.New("min cannot be greater than max")
	}

	return min, max, nil
}

func randomInt64(min int64, max int64) (int64, error) {
	delta := max - min + 1
	if delta <= 0 {
		return 0, errors.New("range is too large")
	}

	limit := big.NewInt(delta)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return 0, err
	}

	return value.Int64() + min, nil
}

func pickWinningPositions(probabilities []float64, winnersCount int) ([]int, error) {
	positionsCount := len(probabilities)
	if winnersCount < 0 {
		return nil, errors.New("winners_count cannot be negative")
	}
	if winnersCount > positionsCount {
		return nil, errors.New("winners_count cannot be greater than probabilities length")
	}
	if winnersCount == 0 {
		return []int{}, nil
	}
	if positionsCount == 0 {
		return nil, errors.New("probabilities cannot be empty")
	}

	indices := make([]int, positionsCount)
	weights := make([]float64, positionsCount)
	positiveWeightsCount := 0

	for idx, probability := range probabilities {
		if math.IsNaN(probability) || math.IsInf(probability, 0) {
			return nil, errors.New("probabilities must be finite numbers")
		}
		if probability < 0 {
			return nil, errors.New("probabilities must be non-negative")
		}
		if probability > 0 {
			positiveWeightsCount++
		}

		indices[idx] = idx
		weights[idx] = probability
	}

	if winnersCount > positiveWeightsCount {
		return nil, errors.New("winners_count cannot be greater than number of positions with non-zero probability")
	}

	winningPositions := make([]int, 0, winnersCount)
	for i := 0; i < winnersCount; i++ {
		chosenIdx, err := pickWeightedIndex(weights)
		if err != nil {
			return nil, err
		}

		winningPositions = append(winningPositions, indices[chosenIdx]+1)
		indices = append(indices[:chosenIdx], indices[chosenIdx+1:]...)
		weights = append(weights[:chosenIdx], weights[chosenIdx+1:]...)
	}

	return winningPositions, nil
}

func pickWeightedIndex(weights []float64) (int, error) {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	if total <= 0 {
		return 0, errors.New("sum of probabilities must be greater than zero")
	}

	randomValue, err := randomFloat64()
	if err != nil {
		return 0, err
	}

	target := randomValue * total
	cumulative := 0.0
	for idx, weight := range weights {
		cumulative += weight
		if target < cumulative {
			return idx, nil
		}
	}

	return len(weights) - 1, nil
}

func randomFloat64() (float64, error) {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return 0, err
	}

	value := binary.BigEndian.Uint64(buffer[:]) >> 11
	const denominator = 1 << 53
	return float64(value) / float64(denominator), nil
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func getenv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	}
	return fallback
}
