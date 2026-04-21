package repository

import "errors"

var (
	ErrInsufficientBalance       = errors.New("insufficient balance")
	ErrRoomIsFull                = errors.New("room is full")
	ErrMaxSeatsExceeded          = errors.New("max seats per user exceeded (50% rule)")
	ErrActiveReservationNotFound = errors.New("active reservation not found")
	ErrInvalidAmount             = errors.New("invalid amount")
	ErrRoundArchived             = errors.New("round already archived")
	ErrRoomNotFound              = errors.New("room not found")
	ErrWrongGameServer           = errors.New("room belongs to another game server")
	ErrSeatAlreadyTaken          = errors.New("seat already taken")
	ErrParticipantNotFound       = errors.New("participant not found")
	ErrParticipantAccessDenied   = errors.New("participant does not belong to user")
	ErrGameAlreadyStarted        = errors.New("game already started")
	ErrRoundAlreadyFinalized     = errors.New("round already finalized")
	ErrRoundNotJoinable          = errors.New("round is not joinable")
	ErrBoostDisabled             = errors.New("boost is disabled for room")
	ErrBoostAlreadyPurchased     = errors.New("boost already purchased")
	ErrInvalidSeatNumber         = errors.New("invalid seat number")
)
