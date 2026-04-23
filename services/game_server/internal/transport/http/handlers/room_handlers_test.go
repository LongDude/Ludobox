package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"game_server/internal/app"
	"game_server/internal/domain"
	"game_server/internal/repository"
	"game_server/internal/service"
	"game_server/internal/transport/dto"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestRouteRoomID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		route      string
		path       string
		headers    map[string]string
		wantStatus int
		wantRoomID int64
	}{
		{
			name:       "path only",
			route:      "/rooms/:roomID/probe",
			path:       "/rooms/123/probe",
			wantStatus: http.StatusOK,
			wantRoomID: 123,
		},
		{
			name:       "matching path header and query",
			route:      "/rooms/:roomID/probe",
			path:       "/rooms/123/probe?room_id=123",
			headers:    map[string]string{"X-Game-Room-ID": "123", "X-Room-ID": "123"},
			wantStatus: http.StatusOK,
			wantRoomID: 123,
		},
		{
			name:       "missing path room id",
			route:      "/probe",
			path:       "/probe",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid path room id",
			route:      "/rooms/:roomID/probe",
			path:       "/rooms/not-a-number/probe",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "mismatched game room header",
			route:      "/rooms/:roomID/probe",
			path:       "/rooms/123/probe",
			headers:    map[string]string{"X-Game-Room-ID": "456"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid room header",
			route:      "/rooms/:roomID/probe",
			path:       "/rooms/123/probe",
			headers:    map[string]string{"X-Room-ID": "abc"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "mismatched query room id",
			route:      "/rooms/:roomID/probe",
			path:       "/rooms/123/probe?room_id=456",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET(tt.route, func(ctx *gin.Context) {
				roomID, err := routeRoomID(ctx)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
					return
				}

				ctx.JSON(http.StatusOK, gin.H{"room_id": roomID})
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				wantBody := fmt.Sprintf(`"room_id":%d`, tt.wantRoomID)
				if !strings.Contains(rec.Body.String(), wantBody) {
					t.Fatalf("body = %s, want to contain %s", rec.Body.String(), wantBody)
				}
			}
		})
	}
}

type joinHandlerRepo struct {
	roomInfo          *domain.RoomInfo
	rounds            map[int64]*domain.Round
	participants      map[int64]*domain.RoundParticipant
	balances          map[int64]int64
	nicknames         map[int64]string
	ratings           map[int64]int64
	nextParticipantID int64
}

func newJoinHandlerRepo() *joinHandlerRepo {
	roundID := int64(10)
	status := "waiting"

	repo := &joinHandlerRepo{
		roomInfo: &domain.RoomInfo{
			Room: &domain.Room{
				RoomID:         1,
				ConfigID:       1,
				ServerID:       1,
				Status:         domain.RoomStatusOpen,
				CurrentPlayers: 1,
			},
			Config: &domain.RoomConfig{
				ConfigID:          1,
				Capacity:          4,
				RegistrationPrice: 100,
				Time:              60,
				RoundTime:         60,
				MinUsers:          2,
			},
			CurrentRoundID:          &roundID,
			CurrentRoundStatus:      &status,
			ActiveParticipantsCount: 1,
		},
		rounds: map[int64]*domain.Round{
			roundID: {
				RoundsID:  roundID,
				RoomID:    1,
				Status:    "waiting",
				CreatedAt: time.Now(),
			},
		},
		participants: map[int64]*domain.RoundParticipant{
			1: {
				RoundParticipantID: 1,
				UserID:             900,
				RoundsID:           roundID,
				NumberInRoom:       1,
			},
		},
		balances:          map[int64]int64{100: 1000, 900: 1000},
		nicknames:         map[int64]string{100: "alice", 900: "other"},
		ratings:           map[int64]int64{100: 777, 900: 0},
		nextParticipantID: 2,
	}

	return repo
}

func (r *joinHandlerRepo) cloneRoomInfo() *domain.RoomInfo {
	roomCopy := *r.roomInfo.Room
	configCopy := *r.roomInfo.Config

	var roundID *int64
	if r.roomInfo.CurrentRoundID != nil {
		value := *r.roomInfo.CurrentRoundID
		roundID = &value
	}

	var roundStatus *string
	if r.roomInfo.CurrentRoundStatus != nil {
		value := *r.roomInfo.CurrentRoundStatus
		roundStatus = &value
	}

	activeCount := 0
	if roundID != nil {
		activeCount, _ = r.GetActiveParticipantsCount(context.Background(), *roundID)
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

func (r *joinHandlerRepo) participantCopy(participant *domain.RoundParticipant) domain.RoundParticipant {
	copyParticipant := *participant
	if nickname, ok := r.nicknames[copyParticipant.UserID]; ok {
		copyParticipant.NickName = stringPtr(nickname)
	}
	if rating, ok := r.ratings[copyParticipant.UserID]; ok {
		copyParticipant.Rating = int64Ptr(rating)
	}
	return copyParticipant
}

func (r *joinHandlerRepo) InTransaction(ctx context.Context, fn func(ts repository.TransactionScope) error) error {
	return fn(r)
}

func (r *joinHandlerRepo) GetParticipantByID(ctx context.Context, participantID int64) (*domain.RoundParticipant, error) {
	participant, ok := r.participants[participantID]
	if !ok {
		return nil, repository.ErrParticipantNotFound
	}
	copyParticipant := r.participantCopy(participant)
	return &copyParticipant, nil
}

func (r *joinHandlerRepo) GetParticipantsByRoundID(ctx context.Context, roundID int64) ([]domain.RoundParticipant, error) {
	result := make([]domain.RoundParticipant, 0)
	for _, participant := range r.participants {
		if participant.RoundsID != roundID || participant.ExitRoomAt != nil {
			continue
		}
		result = append(result, r.participantCopy(participant))
	}
	return result, nil
}

func (r *joinHandlerRepo) GetActiveParticipantsByRoomAndUser(ctx context.Context, roomID, userID int64) ([]domain.RoundParticipant, error) {
	if roomID != r.roomInfo.Room.RoomID || r.roomInfo.CurrentRoundID == nil {
		return nil, nil
	}

	result := make([]domain.RoundParticipant, 0)
	for _, participant := range r.participants {
		if participant.RoundsID == *r.roomInfo.CurrentRoundID && participant.UserID == userID && participant.ExitRoomAt == nil {
			result = append(result, r.participantCopy(participant))
		}
	}
	return result, nil
}

func (r *joinHandlerRepo) CreateRoomEvent(ctx context.Context, roomID int64, roundID *int64, eventType string, eventData json.RawMessage) error {
	return nil
}

func (r *joinHandlerRepo) ListRecentRoomEvents(ctx context.Context, roomID int64, limit int) ([]domain.RoomEvent, error) {
	return nil, nil
}

func (r *joinHandlerRepo) GetRoomsByServerID(ctx context.Context, serverID int64) ([]domain.Room, error) {
	if serverID != r.roomInfo.Room.ServerID {
		return nil, nil
	}
	return []domain.Room{*r.roomInfo.Room}, nil
}

func (r *joinHandlerRepo) GetRoom(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	if roomID != r.roomInfo.Room.RoomID {
		return nil, nil
	}
	return r.cloneRoomInfo(), nil
}

func (r *joinHandlerRepo) GetRoomConfig(ctx context.Context, configID int64) (*domain.RoomConfig, error) {
	if configID != r.roomInfo.Config.ConfigID {
		return nil, repository.ErrRoomNotFound
	}
	configCopy := *r.roomInfo.Config
	return &configCopy, nil
}

func (r *joinHandlerRepo) GetCurrentRoundByRoomID(ctx context.Context, roomID int64) (*int64, error) {
	if roomID != r.roomInfo.Room.RoomID {
		return nil, nil
	}
	return r.roomInfo.CurrentRoundID, nil
}

func (r *joinHandlerRepo) GetRoundInfo(ctx context.Context, roundID int64) (*domain.Round, error) {
	round, ok := r.rounds[roundID]
	if !ok {
		return nil, repository.ErrRoundArchived
	}
	copyRound := *round
	return &copyRound, nil
}

func (r *joinHandlerRepo) GetRoomForUpdate(ctx context.Context, roomID int64) (*domain.RoomInfo, error) {
	if roomID != r.roomInfo.Room.RoomID {
		return nil, repository.ErrRoomNotFound
	}
	return r.cloneRoomInfo(), nil
}

func (r *joinHandlerRepo) CountUserActiveParticipants(ctx context.Context, roundID, userID int64) (int, error) {
	count := 0
	for _, participant := range r.participants {
		if participant.RoundsID == roundID && participant.UserID == userID && participant.ExitRoomAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *joinHandlerRepo) IsSeatOccupied(ctx context.Context, roundID int64, numberInRoom int) (bool, error) {
	for _, participant := range r.participants {
		if participant.RoundsID == roundID && participant.NumberInRoom == numberInRoom && participant.ExitRoomAt == nil {
			return true, nil
		}
	}
	return false, nil
}

func (r *joinHandlerRepo) SetRoomCurrentPlayers(ctx context.Context, roomID int64, currentPlayers int) error {
	r.roomInfo.Room.CurrentPlayers = currentPlayers
	return nil
}

func (r *joinHandlerRepo) GetBalanceLocked(ctx context.Context, userID int64) (int64, error) {
	return r.balances[userID], nil
}

func (r *joinHandlerRepo) UpdateBalance(ctx context.Context, userID int64, delta int64) error {
	r.balances[userID] += delta
	return nil
}

func (r *joinHandlerRepo) CreateParticipant(ctx context.Context, userID, roundID int64, numberInRoom int) (int64, error) {
	id := r.nextParticipantID
	r.nextParticipantID++
	r.participants[id] = &domain.RoundParticipant{
		RoundParticipantID: id,
		UserID:             userID,
		RoundsID:           roundID,
		NumberInRoom:       numberInRoom,
	}
	return id, nil
}

func (r *joinHandlerRepo) GetParticipantUserID(ctx context.Context, participantID int64) (int64, error) {
	participant, ok := r.participants[participantID]
	if !ok {
		return 0, repository.ErrParticipantNotFound
	}
	return participant.UserID, nil
}

func (r *joinHandlerRepo) UpdateParticipantBoost(ctx context.Context, participantID int64, boost int) error {
	return nil
}

func (r *joinHandlerRepo) MarkParticipantExited(ctx context.Context, participantID int64) error {
	now := time.Now()
	r.participants[participantID].ExitRoomAt = &now
	return nil
}

func (r *joinHandlerRepo) UpdateWinningMoney(ctx context.Context, participantID int64, amount int64) error {
	r.participants[participantID].WinningMoney = amount
	return nil
}

func (r *joinHandlerRepo) ApplyUserRatingReward(ctx context.Context, reward domain.UserRatingReward) error {
	r.ratings[reward.UserID] += reward.Delta
	return nil
}

func (r *joinHandlerRepo) ReserveEntry(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	return 1, nil
}

func (r *joinHandlerRepo) ReserveBoost(ctx context.Context, participantID int64, amount int64, expiresAt time.Time) (int64, error) {
	return 1, nil
}

func (r *joinHandlerRepo) ReleaseAllReservations(ctx context.Context, participantID int64) (int64, error) {
	return 0, nil
}

func (r *joinHandlerRepo) ReleaseBoostReservations(ctx context.Context, participantID int64) (int64, error) {
	return 0, nil
}

func (r *joinHandlerRepo) CommitReservations(ctx context.Context, participantID int64) (int64, error) {
	return 0, nil
}

func (r *joinHandlerRepo) ArchiveRound(ctx context.Context, roundID int64) error {
	return nil
}

func (r *joinHandlerRepo) CreateRound(ctx context.Context, roomID int64) (int64, error) {
	return 0, nil
}

func (r *joinHandlerRepo) UpdateRoundStatus(ctx context.Context, roundID int64, status string) error {
	if round, ok := r.rounds[roundID]; ok {
		round.Status = status
	}
	return nil
}

func (r *joinHandlerRepo) GetActiveParticipantsCount(ctx context.Context, roundID int64) (int, error) {
	count := 0
	for _, participant := range r.participants {
		if participant.RoundsID == roundID && participant.ExitRoomAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *joinHandlerRepo) FindFreeNumberInRoom(ctx context.Context, roundID int64, capacity int) (int, error) {
	for seat := 1; seat <= capacity; seat++ {
		occupied, _ := r.IsSeatOccupied(ctx, roundID, seat)
		if !occupied {
			return seat, nil
		}
	}
	return 0, repository.ErrRoomIsFull
}

func (r *joinHandlerRepo) GetRoundStatus(ctx context.Context, roundID int64) (string, error) {
	round, ok := r.rounds[roundID]
	if !ok {
		return "", repository.ErrRoundArchived
	}
	return round.Status, nil
}

func TestJoinRoomReturnsNicknameAndRating(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newJoinHandlerRepo()
	logger := logrus.New()
	events := service.NewEventsService(nil, logger)
	timer := service.NewTimerService(repo, events, logger)
	roomService := service.NewRoomService(repo, logger, 1, nil, "")
	roomService.SetTimerService(timer)

	testApp := &app.App{
		RoomService:   roomService,
		EventsService: events,
		TimerService:  timer,
		Logger:        logger,
	}

	router := gin.New()
	router.POST("/rooms/:roomID/join", func(ctx *gin.Context) { JoinRoom(ctx, testApp) })

	req := httptest.NewRequest(http.MethodPost, "/rooms/1/join", nil)
	req.Header.Set("X-Authenticated-User", "100")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"nickname":"alice"`) {
		t.Fatalf("response does not contain nickname: %s", body)
	}
	if !strings.Contains(body, `"rating":777`) {
		t.Fatalf("response does not contain rating: %s", body)
	}

	var response dto.JoinRoomResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Nickname == nil || *response.Nickname != "alice" {
		t.Fatalf("unexpected nickname: %#v", response.Nickname)
	}
	if response.Rating == nil || *response.Rating != 777 {
		t.Fatalf("unexpected rating: %#v", response.Rating)
	}
}

func stringPtr(value string) *string {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
