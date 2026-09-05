package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
)

type DeviceService struct {
	devices *repositories.DeviceRepository
}

func NewDeviceService(devices *repositories.DeviceRepository) *DeviceService {
	return &DeviceService{devices: devices}
}

func (s *DeviceService) List(f repositories.DeviceFilter, p repositories.Pagination) ([]models.VPNDevice, int64, error) {
	return s.devices.List(f, p)
}

func (s *DeviceService) Get(id uuid.UUID) (*models.VPNDevice, error) {
	device, err := s.devices.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("device %w", ErrNotFound)
		}
		return nil, err
	}
	return device, nil
}

type UpdateDeviceInput struct {
	Name   *string
	Status *models.DeviceStatus
}

func (s *DeviceService) Update(id uuid.UUID, input UpdateDeviceInput) (*models.VPNDevice, error) {
	device, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		device.Name = *input.Name
	}
	if input.Status != nil {
		device.Status = *input.Status
	}
	if err := s.devices.Update(device); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *DeviceService) Disable(id uuid.UUID) (*models.VPNDevice, error) {
	status := models.DeviceStatusDisabled
	return s.Update(id, UpdateDeviceInput{Status: &status})
}

func (s *DeviceService) Revoke(id uuid.UUID) (*models.VPNDevice, error) {
	status := models.DeviceStatusRevoked
	return s.Update(id, UpdateDeviceInput{Status: &status})
}

func (s *DeviceService) Delete(id uuid.UUID) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.devices.Delete(id)
}

func (s *DeviceService) CountActive() (int64, error) {
	return s.devices.CountActive()
}
