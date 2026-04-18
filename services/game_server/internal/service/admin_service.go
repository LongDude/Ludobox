package service

import (
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type AdminService interface {
}

type adminService struct {
	adminRepository repository.AdminRepository
	logger          *logrus.Logger
}

func NewAdminService(adminRepository repository.AdminRepository, logger *logrus.Logger) AdminService {
	return &adminService{
		adminRepository: adminRepository,
		logger:          logger,
	}
}
