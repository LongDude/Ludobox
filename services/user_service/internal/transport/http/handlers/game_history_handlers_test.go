package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

type fakeGameHistoryService struct {
	userID int
	params domain.GameHistoryParams
	called bool
	result domain.ListResponse[domain.GameHistoryItem]
	err    error
}

func (s *fakeGameHistoryService) GetUserGameHistory(ctx context.Context, userID int, params domain.GameHistoryParams) (domain.ListResponse[domain.GameHistoryItem], error) {
	s.userID = userID
	s.params = params
	s.called = true
	return s.result, s.err
}

func TestGetUserGameHistoryRequiresAuthenticatedUserHeader(t *testing.T) {
	recorder, svc := performGameHistoryRequest(t, "/user/history/games", "", &fakeGameHistoryService{})

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if svc.called {
		t.Fatal("service should not be called without authenticated user header")
	}
}

func TestGetUserGameHistoryRejectsInvalidQueryParams(t *testing.T) {
	recorder, svc := performGameHistoryRequest(t, "/user/history/games?page=abc", "42", &fakeGameHistoryService{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if svc.called {
		t.Fatal("service should not be called when query params are invalid")
	}
}

func TestGetUserGameHistoryReturnsHistory(t *testing.T) {
	joinedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	finishedAt := joinedAt.Add(2 * time.Minute)
	svc := &fakeGameHistoryService{
		result: domain.ListResponse[domain.GameHistoryItem]{
			Items: []domain.GameHistoryItem{
				{
					RoundID:            77,
					RoomID:             9,
					GameID:             3,
					GameName:           "Dice",
					RoundStatus:        "finished",
					Result:             "won",
					ReservedSeats:      []int{2, 4, 6},
					WinningSeats:       []int{4},
					ReservedSeatsCount: 3,
					WinningSeatsCount:  1,
					EntryFee:           300,
					BoostFee:           25,
					TotalSpent:         325,
					WinningMoney:       400,
					NetResult:          75,
					JoinedAt:           joinedAt,
					FinishedAt:         &finishedAt,
				},
			},
			Total: 1,
		},
	}

	recorder, _ := performGameHistoryRequest(
		t,
		"/user/history/games?page=2&page_size=5&game_id=3&room_id=9&status=won&date_from=2026-04-01&date_to=2026-04-22",
		"42",
		svc,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !svc.called {
		t.Fatal("service was not called")
	}
	if svc.userID != 42 {
		t.Fatalf("userID = %d, want 42", svc.userID)
	}
	if svc.params.Page != 2 || svc.params.PageSize != 5 {
		t.Fatalf("pagination = page %d size %d, want page 2 size 5", svc.params.Page, svc.params.PageSize)
	}
	if svc.params.GameID == nil || *svc.params.GameID != 3 {
		t.Fatalf("game_id filter = %v, want 3", svc.params.GameID)
	}
	if svc.params.RoomID == nil || *svc.params.RoomID != 9 {
		t.Fatalf("room_id filter = %v, want 9", svc.params.RoomID)
	}
	if svc.params.Status != "won" {
		t.Fatalf("status filter = %q, want won", svc.params.Status)
	}
	if svc.params.DateFrom == nil || svc.params.DateTo == nil {
		t.Fatal("date filters should be parsed")
	}

	var response presenters.GameHistoryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Total != 1 || response.Page != 2 || response.PageSize != 5 {
		t.Fatalf("response meta = total %d page %d size %d", response.Total, response.Page, response.PageSize)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(response.Items))
	}
	item := response.Items[0]
	if item.Result != "won" || item.NetResult != 75 || item.EntryFee != 300 || item.BoostFee != 25 || item.TotalSpent != 325 {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.ReservedSeatsCount != 3 || item.WinningSeatsCount != 1 {
		t.Fatalf("unexpected seat counts: %+v", item)
	}
	if len(item.ReservedSeats) != 3 || len(item.WinningSeats) != 1 {
		t.Fatalf("unexpected seat lists: %+v", item)
	}
}

func performGameHistoryRequest(t *testing.T, target string, authenticatedUser string, svc *fakeGameHistoryService) (*httptest.ResponseRecorder, *fakeGameHistoryService) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/user/history/games", func(ctx *gin.Context) {
		GetUserGameHistory(ctx, &app.App{GameHistoryService: svc})
	})

	request := httptest.NewRequest(http.MethodGet, target, nil)
	if authenticatedUser != "" {
		request.Header.Set("X-Authenticated-User", authenticatedUser)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder, svc
}
