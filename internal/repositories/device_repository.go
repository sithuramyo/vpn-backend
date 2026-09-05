package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

type DeviceRepository struct {
	db *gorm.DB
}

type DeviceFilter struct {
	Status   string
	Platform string
	UserID   *uuid.UUID
}

func (r *DeviceRepository) List(f DeviceFilter, p Pagination) ([]models.VPNDevice, int64, error) {
	page, pageSize, offset := p.normalize()

	q := r.db.Model(&models.VPNDevice{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Platform != "" {
		q = q.Where("platform = ?", f.Platform)
	}
	if f.UserID != nil {
		q = q.Where("vpn_user_id = ?", *f.UserID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var devices []models.VPNDevice
	err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Preload("User").Find(&devices).Error
	_ = page
	return devices, total, err
}

func (r *DeviceRepository) FindByID(id uuid.UUID) (*models.VPNDevice, error) {
	var device models.VPNDevice
	if err := r.db.Preload("User").First(&device, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) Create(device *models.VPNDevice) error {
	return r.db.Create(device).Error
}

func (r *DeviceRepository) Update(device *models.VPNDevice) error {
	return r.db.Save(device).Error
}

func (r *DeviceRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.VPNDevice{}, "id = ?", id).Error
}

func (r *DeviceRepository) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&models.VPNDevice{}).Where("status = ?", models.DeviceStatusActive).Count(&count).Error
	return count, err
}
