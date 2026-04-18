package presenters

type TestResponse struct {
	Pong string `json:"pong" binding:"required" validate:"required"`
}

type UserResponse struct {
}
