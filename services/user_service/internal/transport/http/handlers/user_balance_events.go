package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"user_service/internal/app"
	"user_service/internal/repository"
	"user_service/internal/service"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

const userBalanceEventsHeartbeatInterval = 25 * time.Second

// SubscribeUserBalanceEvents godoc
// @Summary Subscribe current user balance events
// @Description Streams balance changes for the authenticated user.
// @Tags User
// @Accept json
// @Produce text/event-stream
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Success 200 {object} service.UserBalanceEvent
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user/balance/events [get]
func SubscribeUserBalanceEvents(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	if a.UserBalanceEvents == nil {
		ctx.JSON(http.StatusServiceUnavailable, presenters.Error(fmt.Errorf("user balance events are unavailable")))
		return
	}

	user, err := a.UserService.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrorUserNotFound) {
			user, err = a.UserService.CreateUserByID(ctx, userID)
		}
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("load user failed: %w", err)))
			return
		}
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

	events, unsubscribe := a.UserBalanceEvents.Subscribe(ctx.Request.Context(), int64(userID))
	defer unsubscribe()

	if err := writeUserBalanceEvent(ctx.Writer, service.UserBalanceEvent{
		Type:      "user_balance_snapshot",
		Action:    "snapshot",
		UserID:    int64(user.ID),
		Balance:   int64(user.Balance),
		Timestamp: time.Now().UTC(),
	}); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(userBalanceEventsHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeUserBalanceEvent(ctx.Writer, event); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(ctx.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-a.UserBalanceEvents.Done():
			return
		case <-ctx.Request.Context().Done():
			return
		}
	}
}

func writeUserBalanceEvent(w http.ResponseWriter, event service.UserBalanceEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}
