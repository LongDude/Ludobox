package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"game_server/internal/transport/dto"

	"github.com/gin-gonic/gin"
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
