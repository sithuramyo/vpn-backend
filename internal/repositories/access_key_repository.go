package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

type AccessKeyRepository struct {
	db *gorm.DB
}

type AccessKeyFilter struct {
	Status   string
	UserID   *uuid.UUID
	ServerID *uuid.UUID
}

func (r *AccessKeyRepository) List(f AccessKeyFilter, p Pagination) ([]models.AccessKey, int64, error) {
	page, pageSize, offset := p.normalize()

	q := r.db.Model(&models.AccessKey{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.UserID != nil {
		q = q.Where("vpn_user_id = ?", *f.UserID)
	}
	if f.ServerID != nil {
		q = q.Where("vpn_server_id = ?", *f.ServerID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var keys []models.AccessKey
	err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).
		Preload("User").Preload("Server").
		Find(&keys).Error
	_ = page
	return keys, total, err
}

func (r *AccessKeyRepository) FindByID(id uuid.UUID) (*models.AccessKey, error) {
	var key models.AccessKey
	if err := r.db.Preload("User").Preload("Server").First(&key, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *AccessKeyRepository) Create(key *models.AccessKey) error {
	return r.db.Create(key).Error
}

func (r *AccessKeyRepository) Update(key *models.AccessKey) error {
	return r.db.Save(key).Error
}

func (r *AccessKeyRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.AccessKey{}, "id = ?", id).Error
}

func (r *AccessKeyRepository) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&models.AccessKey{}).Where("status = ?", models.AccessKeyStatusActive).Count(&count).Error
	return count, err
}

func (r *AccessKeyRepository) FindByUser(userID uuid.UUID) ([]models.AccessKey, error) {
	var keys []models.AccessKey
	err := r.db.Where("vpn_user_id = ?", userID).Preload("Server").Find(&keys).Error
	return keys, err
}

func (r *AccessKeyRepository) FindExpired() ([]models.AccessKey, error) {
	var keys []models.AccessKey
	err := r.db.Where("status = ? AND expires_at IS NOT NULL AND expires_at < now()", models.AccessKeyStatusActive).
		Find(&keys).Error
	return keys, err
}

func (r *AccessKeyRepository) SumTrafficByServer(serverID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.Model(&models.AccessKey{}).
		Where("vpn_server_id = ?", serverID).
		Select("COALESCE(SUM(traffic_used_bytes), 0)").
		Scan(&total).Error
	return total, err
}
