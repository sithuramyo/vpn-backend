package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
)

type AdminService struct {
	admins *repositories.AdminRepository
}

func NewAdminService(admins *repositories.AdminRepository) *AdminService {
	return &AdminService{admins: admins}
}

func (s *AdminService) List(p repositories.Pagination) ([]models.Admin, int64, error) {
	return s.admins.List(p)
}

func (s *AdminService) Get(id uuid.UUID) (*models.Admin, error) {
	admin, err := s.admins.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("admin %w", ErrNotFound)
		}
		return nil, err
	}
	return admin, nil
}

type UpdateAdminInput struct {
	Role   *models.AdminRole
	Status *models.AdminStatus
}

func (s *AdminService) Update(id uuid.UUID, input UpdateAdminInput) (*models.Admin, error) {
	admin, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if input.Role != nil {
		admin.Role = *input.Role
	}
	if input.Status != nil {
		admin.Status = *input.Status
	}
	if err := s.admins.Update(admin); err != nil {
		return nil, err
	}
	return admin, nil
}
