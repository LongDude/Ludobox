package repository

import "errors"

var (
	ErrInsufficientBalance       = errors.New("insufficient balance")
	ErrRoomIsFull                = errors.New("room is full")
	ErrMaxSeatsExceeded          = errors.New("max seats per user exceeded (50% rule)")
	ErrActiveReservationNotFound = errors.New("active reservation not found")
	ErrInvalidAmount             = errors.New("invalid amount")
	ErrRoundArchived             = errors.New("round already archived")
)
