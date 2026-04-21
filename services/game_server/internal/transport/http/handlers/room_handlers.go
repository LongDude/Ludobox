package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"game_server/internal/app"
	"game_server/internal/transport/dto"

	"github.com/gin-gonic/gin"
)

// authenticatedUserID извлекает userID из заголовка X-Authenticated-User
func authenticatedUserID(ctx *gin.Context) (int64, error) {
	userIDHeader := strings.TrimSpace(ctx.GetHeader("X-Authenticated-User"))
	if userIDHeader == "" {
		return 0, fmt.Errorf("missing X-Authenticated-User header")
	}

	userID, err := strconv.ParseInt(userIDHeader, 10, 64)
	if err != nil || userID <= 0 {
		return 0, fmt.Errorf("invalid X-Authenticated-User header")
	}

	return userID, nil
}

// ParamInt64 парсит параметр пути как int64
func paramInt64(ctx *gin.Context, key string) (int64, error) {
	value := ctx.Param(key)
	return strconv.ParseInt(value, 10, 64)
}

// JoinRoom godoc
// @Summary Join room
// @Description User joins a room and gets assigned a seat
// @Tags Rooms
// @Accept json
// @Produce json
// @Param request body dto.JoinRoomRequest true "Join request"
// @Success 200 {object} dto.JoinRoomResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /rooms/join [post]
func JoinRoom(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	var req dto.JoinRoomRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Вызываем RoomService.JoinRoom
	participantID, err := a.RoomService.JoinRoom(ctx.Request.Context(), userID, req.RoomID)
	if err != nil {
		if err.Error() == "room is full" {
			ctx.JSON(http.StatusConflict, dto.ErrorResponse{Error: "Room is full", Code: "ROOM_FULL"})
			return
		}
		if err.Error() == "insufficient balance" {
			ctx.JSON(http.StatusPaymentRequired, dto.ErrorResponse{Error: "Insufficient balance", Code: "INSUFFICIENT_BALANCE"})
			return
		}
		a.Logger.Errorf("Error joining room: %v", err)
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Получаем информацию о раунде и комнате
	roomInfo, err := a.RoomService.GetRoomInfo(ctx.Request.Context(), req.RoomID)
	if err != nil {
		a.Logger.Errorf("Error getting room info: %v", err)
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Публикуем событие присоединения
	if roomInfo.CurrentRoundID != nil {
		a.EventsService.PublishPlayerJoined(
			ctx.Request.Context(),
			*roomInfo.CurrentRoundID,
			participantID,
			1, // TODO: получить правильный номер места
			roomInfo.ActiveParticipantsCount,
		)
	}

	response := dto.JoinRoomResponse{
		ParticipantID:  participantID,
		RoomCapacity:   int64(roomInfo.Config.Capacity),
		CurrentPlayers: roomInfo.ActiveParticipantsCount,
		MinPlayers:     roomInfo.Config.MinUsers,
		EntryPrice:     roomInfo.Config.RegistrationPrice,
		RoundStatus:    string(roomInfo.Room.Status),
	}

	if roomInfo.CurrentRoundID != nil {
		response.RoundID = *roomInfo.CurrentRoundID
	}

	ctx.JSON(http.StatusOK, response)
}

// JoinRoomWithSeat godoc
// @Summary Join room with specific seat
// @Description User joins a room and selects a specific seat
// @Tags Rooms
// @Accept json
// @Produce json
// @Param request body dto.JoinRoomWithSeatRequest true "Join with seat request"
// @Success 200 {object} dto.JoinRoomResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /rooms/join-seat [post]
func JoinRoomWithSeat(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	var req dto.JoinRoomWithSeatRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// TODO: Реализовать JoinRoomWithSeat в RoomService
	ctx.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Not implemented yet"})
}

// PurchaseBoost godoc
// @Summary Purchase boost
// @Description User purchases a boost for their seat
// @Tags Rooms
// @Accept json
// @Produce json
// @Param roundParticipantID path int64 true "Round participant ID"
// @Param request body dto.PurchaseBoostRequest true "Purchase boost request"
// @Success 200 {object} dto.PurchaseBoostResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /rooms/participants/{roundParticipantID}/boost [post]
func PurchaseBoost(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	participantID, err := paramInt64(ctx, "participantID")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid participant ID"})
		return
	}

	var req dto.PurchaseBoostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Получаем информацию об участнике для проверки
	participant, err := a.RoomService.GetParticipantInfo(ctx.Request.Context(), participantID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Participant not found"})
		return
	}

	// Получаем конфиг комнаты для цены буста
	roomInfo, err := a.RoomService.GetRoomInfoByRound(ctx.Request.Context(), participant.RoundsID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Покупаем буст
	err = a.RoomService.PurchaseBoost(
		ctx.Request.Context(),
		participantID,
		userID,
		req.BoostValue,
		roomInfo.Config.BoostPrice,
	)
	if err != nil {
		if err.Error() == "insufficient balance" {
			ctx.JSON(http.StatusPaymentRequired, dto.ErrorResponse{Error: "Insufficient balance", Code: "INSUFFICIENT_BALANCE"})
			return
		}
		a.Logger.Errorf("Error purchasing boost: %v", err)
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Публикуем событие
	a.EventsService.PublishBoostPurchased(
		ctx.Request.Context(),
		participant.RoundsID,
		participantID,
		int(req.BoostValue),
	)

	ctx.JSON(http.StatusOK, dto.PurchaseBoostResponse{
		Success:    true,
		BoostPower: int(req.BoostValue),
		BoostCost:  roomInfo.Config.BoostPrice,
	})
}

// CancelBoost godoc
// @Summary Cancel boost
// @Description User cancels their boost
// @Tags Rooms
// @Accept json
// @Produce json
// @Param roundParticipantID path int64 true "Round participant ID"
// @Success 200 {object} dto.CancelBoostResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /rooms/participants/{roundParticipantID}/boost [delete]
func CancelBoost(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	participantID, err := paramInt64(ctx, "participantID")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid participant ID"})
		return
	}

	// Получаем информацию об участнике
	participant, err := a.RoomService.GetParticipantInfo(ctx.Request.Context(), participantID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Participant not found"})
		return
	}

	// Отменяем буст
	err = a.RoomService.CancelBoost(ctx.Request.Context(), participantID, userID)
	if err != nil {
		a.Logger.Errorf("Error cancelling boost: %v", err)
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Публикуем событие
	a.EventsService.PublishBoostCancelled(ctx.Request.Context(), participant.RoundsID, participantID)

	ctx.JSON(http.StatusOK, dto.CancelBoostResponse{
		Success: true,
	})
}

// LeaveRoom godoc
// @Summary Leave room
// @Description User leaves the room before game starts
// @Tags Rooms
// @Accept json
// @Produce json
// @Param roundParticipantID path int64 true "Round participant ID"
// @Success 200 {object} dto.LeaveRoomResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /rooms/participants/{roundParticipantID}/leave [post]
func LeaveRoom(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	participantID, err := paramInt64(ctx, "participantID")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid participant ID"})
		return
	}

	// Получаем информацию об участнике
	participant, err := a.RoomService.GetParticipantInfo(ctx.Request.Context(), participantID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Participant not found"})
		return
	}

	// Выходим из комнаты
	err = a.RoomService.LeaveRoom(ctx.Request.Context(), participantID, userID)
	if err != nil {
		if err.Error() == "game already started" {
			ctx.JSON(http.StatusConflict, dto.ErrorResponse{Error: "Cannot leave during game", Code: "GAME_STARTED"})
			return
		}
		a.Logger.Errorf("Error leaving room: %v", err)
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Публикуем событие
	// TODO: получить правильное количество активных участников
	a.EventsService.PublishPlayerLeft(ctx.Request.Context(), participant.RoundsID, participantID, participant.NumberInRoom, 0)

	ctx.JSON(http.StatusOK, dto.LeaveRoomResponse{
		Success: true,
	})
}

// GetRoundStatus godoc
// @Summary Get round status
// @Description Get current status of a round
// @Tags Rounds
// @Accept json
// @Produce json
// @Param roundID path int64 true "Round ID"
// @Success 200 {object} dto.RoundStatusResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /rounds/{roundID} [get]
func GetRoundStatus(ctx *gin.Context, a *app.App) {
	roundID, err := paramInt64(ctx, "roundID")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid round ID"})
		return
	}

	// TODO: Получить информацию о раунде из БД
	ctx.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Not implemented yet"})
}

// SubscribeToRoundEvents godoc
// @Summary Subscribe to round events (SSE)
// @Description Subscribe to real-time updates for a round
// @Tags Rounds
// @Param roundID path int64 true "Round ID"
// @Success 200 {object} string "SSE stream"
// @Failure 400 {object} dto.ErrorResponse
// @Router /rounds/{roundID}/events [get]
func SubscribeToRoundEvents(ctx *gin.Context, a *app.App) {
	roundID, err := paramInt64(ctx, "roundID")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid round ID"})
		return
	}

	// Подписываемся на события
	eventChan := a.EventsService.Subscribe(roundID)
	defer a.EventsService.Unsubscribe(roundID, eventChan)

	// Устанавливаем SSE заголовки
	w := ctx.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	ctx.Status(http.StatusOK)
	w.(http.Flusher).Flush()

	// Слушаем события
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-eventChan:
			if event == nil {
				return
			}
			message, err := a.EventsService.EncodeSSEMessage(event)
			if err != nil {
				a.Logger.Errorf("Error encoding SSE message: %v", err)
				continue
			}
			if _, err := fmt.Fprint(w, message); err != nil {
				a.Logger.Debugf("Error writing SSE message: %v", err)
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-ticker.C:
			// Отправляем heartbeat
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-ctx.Request.Context().Done():
			return
		}
	}
}

// InternalStartGame - внутренний эндпоинт для запуска игры (вызывается таймером)
// @Summary Start game (internal)
// @Description Transition round to active state and finalize
// @Tags Internal
// @Accept json
// @Produce json
// @Param roundID path int64 true "Round ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Router /internal/rounds/{roundID}/start [post]
func InternalStartGame(ctx *gin.Context, a *app.App) {
	roundID, err := paramInt64(ctx, "roundID")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid round ID"})
		return
	}

	err = a.RoomService.StartGameRound(ctx.Request.Context(), roundID)
	if err != nil {
		a.Logger.Errorf("Error starting game for round %d: %v", roundID, err)
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Game started",
	})
}

// InternalFinalizeGame - внутренний эндпоинт для завершения игры (вызывается таймером)
// @Summary Finalize game (internal)
// @Description Finish round, select winners, and finalize
// @Tags Internal
// @Accept json
// @Produce json
// @Param roundID path int64 true "Round ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Router /internal/rounds/{roundID}/finalize [post]
func InternalFinalizeGame(ctx *gin.Context, a *app.App) {
	roundID, err := paramInt64(ctx, "roundID")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid round ID"})
		return
	}

	winners, err := a.RoomService.FinalizeGameRound(ctx.Request.Context(), roundID)
	if err != nil {
		if err.Error() == "round already finalized" {
			ctx.JSON(http.StatusConflict, dto.ErrorResponse{Error: "Round already finalized", Code: "ALREADY_FINALIZED"})
			return
		}
		a.Logger.Errorf("Error finalizing game for round %d: %v", roundID, err)
		ctx.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Game finalized",
		"winners": winners,
	})
}
