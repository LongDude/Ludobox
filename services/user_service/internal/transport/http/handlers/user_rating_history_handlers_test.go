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

	"github.com/gin-gonic/gin"
)

type fakeUserRatingService struct {
	user                   *domain.User
	userErr                error
	createErr              error
	history                domain.UserRatingHistory
	historyErr             error
	getUserCalled          bool
	createUserCalled       bool
	getRatingHistoryCalled bool
	userID                 int
	params                 domain.UserRatingHistoryParams
}

func (s *fakeUserRatingService) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	s.getUserCalled = true
	s.userID = id
	return s.user, s.userErr
}

func (s *fakeUserRatingService) CreateUserByID(ctx context.Context, id int) (*domain.User, error) {
	s.createUserCalled = true
	s.userID = id
	return s.user, s.createErr
}

func (s *fakeUserRatingService) UpdateUserByID(ctx context.Context, id int, user *domain.User) (*domain.User, error) {
	return nil, nil
}

func (s *fakeUserRatingService) UpdateUserBalance(ctx context.Context, balanceSum, id int) (*domain.User, error) {
	return nil, nil
}

func (s *fakeUserRatingService) GetUserRatingHistory(ctx context.Context, userID int, params domain.UserRatingHistoryParams) (domain.UserRatingHistory, error) {
	s.getRatingHistoryCalled = true
	s.userID = userID
	s.params = params
	return s.history, s.historyErr
}

func (s *fakeUserRatingService) DeleteUserByID(ctx context.Context, id int) error {
	return nil
}

func TestGetUserRatingHistoryRequiresAuthenticatedUserHeader(t *testing.T) {
	recorder, svc := performUserRatingHistoryRequest(t, "/user/history/rating", "", &fakeUserRatingService{})

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if svc.getRatingHistoryCalled {
		t.Fatal("service should not be called without authenticated user header")
	}
}

func TestGetUserRatingHistoryRejectsInvalidDateRange(t *testing.T) {
	recorder, svc := performUserRatingHistoryRequest(
		t,
		"/user/history/rating?date_from=2026-04-22&date_to=2026-04-01",
		"42",
		&fakeUserRatingService{},
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if svc.getRatingHistoryCalled {
		t.Fatal("service should not be called when query params are invalid")
	}
}

func TestGetUserRatingHistoryReturnsHistory(t *testing.T) {
	createdAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	roundID := int64(77)
	roomID := int64(9)

	svc := &fakeUserRatingService{
		user: &domain.User{ID: 42, Rating: 180, Rank: "bronze"},
		history: domain.UserRatingHistory{
			CurrentRating: 180,
			CurrentRank:   "bronze",
			PeriodChange:  42,
			Items: []domain.UserRatingHistoryPoint{
				{
					HistoryID:   1,
					RoundID:     &roundID,
					RoomID:      &roomID,
					Source:      "round_win",
					Delta:       42,
					RatingAfter: 180,
					Rank:        "bronze",
					CreatedAt:   createdAt,
				},
			},
		},
	}

	recorder, _ := performUserRatingHistoryRequest(
		t,
		"/user/history/rating?date_from=2026-04-01&date_to=2026-04-22",
		"42",
		svc,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !svc.getRatingHistoryCalled {
		t.Fatal("service was not called")
	}
	if svc.params.DateFrom == nil || svc.params.DateTo == nil {
		t.Fatal("date filters should be parsed")
	}

	var response struct {
		CurrentRating int64  `json:"current_rating"`
		CurrentRank   string `json:"current_rank"`
		PeriodChange  int64  `json:"period_change"`
		Items         []struct {
			HistoryID   int64  `json:"history_id"`
			Source      string `json:"source"`
			Delta       int64  `json:"delta"`
			RatingAfter int64  `json:"rating_after"`
			Rank        string `json:"rank"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.CurrentRating != 180 || response.CurrentRank != "bronze" || response.PeriodChange != 42 {
		t.Fatalf("unexpected response meta: %+v", response)
	}
	if len(response.Items) != 1 || response.Items[0].Delta != 42 {
		t.Fatalf("unexpected items: %+v", response.Items)
	}
}

func performUserRatingHistoryRequest(t *testing.T, target string, authenticatedUser string, svc *fakeUserRatingService) (*httptest.ResponseRecorder, *fakeUserRatingService) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/user/history/rating", func(ctx *gin.Context) {
		GetUserRatingHistory(ctx, &app.App{UserService: svc})
	})

	request := httptest.NewRequest(http.MethodGet, target, nil)
	if authenticatedUser != "" {
		request.Header.Set("X-Authenticated-User", authenticatedUser)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder, svc
}
