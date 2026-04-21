package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
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

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/random", randomHandler)

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
