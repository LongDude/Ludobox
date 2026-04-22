package domain

type UserRatingReward struct {
	UserID        int64
	ParticipantID int64
	RoundID       int64
	RoomID        int64
	GameID        int64
	Source        string
	Delta         int64
}
