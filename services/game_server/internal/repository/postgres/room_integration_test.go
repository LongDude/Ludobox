package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"game_server/internal/service"
)

func TestRoomServiceIntegrationSmoke(t *testing.T) {
	connStr := os.Getenv("GAME_SERVER_TEST_DATABASE_URL")
	if connStr == "" {
		t.Skip("GAME_SERVER_TEST_DATABASE_URL is not set")
	}

	db, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}
	defer db.Close()

	repo := NewRoomRepository(db)
	svc := service.NewRoomService(repo, logrus.New(), 1, nil, "")

	if _, err := svc.GetRoomInfo(context.Background(), 1); err != nil {
		t.Fatalf("GetRoomInfo returned error: %v", err)
	}
}
