package http

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMainRouterRegistersRoomScopedRoutesOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MainRouter(router.Group("/api"), nil)

	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	want := []string{
		"POST /api/rooms/:roomID/join",
		"POST /api/rooms/:roomID/join-seat",
		"POST /api/rooms/:roomID/leave",
		"POST /api/rooms/:roomID/participants/:participantID/boost",
		"DELETE /api/rooms/:roomID/participants/:participantID/boost",
		"POST /api/rooms/:roomID/participants/:participantID/leave",
		"GET /api/rooms/:roomID/rounds/:roundID",
		"GET /api/rooms/:roomID/rounds/:roundID/events",
		"POST /api/internal/rounds/:roundID/start",
		"POST /api/internal/rounds/:roundID/finalize",
	}
	for _, route := range want {
		if !routes[route] {
			t.Fatalf("route %q is not registered", route)
		}
	}

	legacy := []string{
		"POST /api/rooms/join",
		"POST /api/rooms/join-seat",
		"POST /api/rooms/leave",
		"POST /api/rooms/participants/:participantID/boost",
		"DELETE /api/rooms/participants/:participantID/boost",
		"POST /api/rooms/participants/:participantID/leave",
		"GET /api/rounds/:roundID",
		"GET /api/rounds/:roundID/events",
	}
	for _, route := range legacy {
		if routes[route] {
			t.Fatalf("legacy route %q is still registered", route)
		}
	}
}
