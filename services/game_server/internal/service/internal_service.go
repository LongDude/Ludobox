package service

import (
	"game_server/internal/repository"

	"github.com/sirupsen/logrus"
)

type InternalService interface {
}

type internalService struct {
	internalRepository repository.InternalRepository
	logger             *logrus.Logger
}

func NewInternalService(internalRepository repository.InternalRepository, logger *logrus.Logger) InternalService {
	return &internalService{
		internalRepository: internalRepository,
		logger:             logger,
	}
}
