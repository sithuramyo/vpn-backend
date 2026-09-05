package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/vpn"
)

type UserService struct {
	users      *repositories.VPNUserRepository
	devices    *repositories.DeviceRepository
	accessKeys *repositories.AccessKeyRepository
	provider   vpn.VPNProvider
}

func NewUserService(users *repositories.VPNUserRepository, devices *repositories.DeviceRepository, accessKeys *repositories.AccessKeyRepository, provider vpn.VPNProvider) *UserService {
	return &UserService{users: users, devices: devices, accessKeys: accessKeys, provider: provider}
}

type CreateUserInput struct {
	Name              string
	Email             string
	ExpiresAt         *time.Time
	TrafficLimitBytes int64
}

func (s *UserService) Create(input CreateUserInput) (*models.VPNUser, error) {
	user := &models.VPNUser{
		Name:              input.Name,
		Email:             input.Email,
		Status:            models.VPNUserStatusActive,
		ExpiresAt:         input.ExpiresAt,
		TrafficLimitBytes: input.TrafficLimitBytes,
	}
	if err := s.users.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) List(f repositories.UserFilter, p repositories.Pagination) ([]models.VPNUser, int64, error) {
	return s.users.List(f, p)
}

func (s *UserService) Get(id uuid.UUID) (*models.VPNUser, error) {
	user, err := s.users.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user %w", ErrNotFound)
		}
		return nil, err
	}
	return user, nil
}

type UpdateUserInput struct {
	Name              *string
	Email             *string
	ExpiresAt         **time.Time
	TrafficLimitBytes *int64
}

func (s *UserService) Update(id uuid.UUID, input UpdateUserInput) (*models.VPNUser, error) {
	user, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.ExpiresAt != nil {
		user.ExpiresAt = *input.ExpiresAt
	}
	if input.TrafficLimitBytes != nil {
		user.TrafficLimitBytes = *input.TrafficLimitBytes
	}

	if err := s.users.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

// Disable sets the user to DISABLED and revokes every active access key and
// device they hold, since a disabled user must lose VPN access immediately.
func (s *UserService) Disable(ctx context.Context, id uuid.UUID) (*models.VPNUser, error) {
	user, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	user.Status = models.VPNUserStatusDisabled
	if err := s.users.Update(user); err != nil {
		return nil, err
	}
	if err := s.revokeAllAccess(ctx, id); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Enable(id uuid.UUID) (*models.VPNUser, error) {
	user, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	user.Status = models.VPNUserStatusActive
	if err := s.users.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	if err := s.revokeAllAccess(ctx, id); err != nil {
		return err
	}
	return s.users.Delete(id)
}

func (s *UserService) revokeAllAccess(ctx context.Context, userID uuid.UUID) error {
	keys, err := s.accessKeys.FindByUser(userID)
	if err != nil {
		return err
	}
	for i := range keys {
		key := &keys[i]
		if key.Status != models.AccessKeyStatusActive {
			continue
		}
		if err := s.provider.RevokeAccessKey(ctx, key.SecretReference); err != nil {
			return fmt.Errorf("revoke access key %s: %w", key.ID, err)
		}
		key.Status = models.AccessKeyStatusRevoked
		if err := s.accessKeys.Update(key); err != nil {
			return err
		}
	}

	devices, _, err := s.devices.List(repositories.DeviceFilter{UserID: &userID}, repositories.Pagination{PageSize: 1000})
	if err != nil {
		return err
	}
	for i := range devices {
		device := &devices[i]
		if device.Status == models.DeviceStatusRevoked {
			continue
		}
		device.Status = models.DeviceStatusRevoked
		if err := s.devices.Update(device); err != nil {
			return err
		}
	}

	return nil
}

// SweepExpired disables any user whose expires_at has passed and revokes
// their access, mirroring Disable's cascade. Intended to run on a periodic
// background ticker.
func (s *UserService) SweepExpired(ctx context.Context) (int, error) {
	expired, err := s.users.FindExpired()
	if err != nil {
		return 0, err
	}
	for i := range expired {
		user := &expired[i]
		user.Status = models.VPNUserStatusExpired
		if err := s.users.Update(user); err != nil {
			return 0, err
		}
		if err := s.revokeAllAccess(ctx, user.ID); err != nil {
			return 0, err
		}
	}
	return len(expired), nil
}

func (s *UserService) CountTotal() (int64, error) { return s.users.Count() }

func (s *UserService) CountActive() (int64, error) {
	return s.users.CountByStatus(models.VPNUserStatusActive)
}
