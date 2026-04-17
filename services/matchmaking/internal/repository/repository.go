package repository

import (
	"errors"
)

var (
	ErrorUserNotFound     = errors.New("user not found")
	ErrorUserAlreadyExist = errors.New("user already exist")
)

type AdminRepository interface {
}
type SessionRepository interface {
}
