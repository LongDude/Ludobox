package domain

import "time"

type ReservationType string

const (
	ReservationTypeEntryFee ReservationType = "entry_fee"
	ReservationTypeBoost    ReservationType = "boost"
)

func (rt ReservationType) IsValid() bool {
	return rt == ReservationTypeEntryFee || rt == ReservationTypeBoost
}

type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "active"
	ReservationStatusReleased  ReservationStatus = "released"
	ReservationStatusCommitted ReservationStatus = "committed"
)

type Reservation struct {
	RoundParticipantID int64             `json:"round_participants_id"`
	ReservationType    ReservationType   `json:"reservation_type"`
	Amount             int64             `json:"amount"`
	Status             ReservationStatus `json:"status"`
	ExpiresAt          time.Time         `json:"expires_at"`
	CreatedAt          time.Time         `json:"created_at"`
	ArchivedAt         *time.Time        `json:"archived_at"`
}
