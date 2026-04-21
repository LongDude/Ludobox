package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"user_service/internal/app"
	"user_service/internal/service"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

const adminEventsHeartbeatInterval = 25 * time.Second

// SubscribeAdminEvents godoc
// @Summary Subscribe admin events
// @Description Streams admin resource change events for games, configs, rooms and servers.
// @Tags Admin Events
// @Accept json
// @Produce text/event-stream
// @Param Authorization header string true "Authorization Token"
// @Success 200 {object} service.AdminEvent
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 403 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /admin/events [get]
func SubscribeAdminEvents(ctx *gin.Context, a *app.App) {
	if a.AdminEvents == nil {
		ctx.JSON(http.StatusServiceUnavailable, presenters.Error(fmt.Errorf("admin events are unavailable")))
		return
	}

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("streaming is unsupported")))
		return
	}

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")
	ctx.Status(http.StatusOK)

	events, unsubscribe := a.AdminEvents.Subscribe(ctx.Request.Context())
	defer unsubscribe()

	if err := writeAdminEvent(ctx.Writer, service.AdminEvent{
		Type:      "admin_connected",
		Action:    "connected",
		Timestamp: time.Now().UTC(),
	}); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(adminEventsHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeAdminEvent(ctx.Writer, event); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(ctx.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-a.AdminEvents.Done():
			return
		case <-ctx.Request.Context().Done():
			return
		}
	}
}

func writeAdminEvent(w http.ResponseWriter, event service.AdminEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}
