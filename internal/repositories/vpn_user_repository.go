package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

type VPNUserRepository struct {
	db *gorm.DB
}

type UserFilter struct {
	Status string
	Search string
}

func (r *VPNUserRepository) List(f UserFilter, p Pagination) ([]models.VPNUser, int64, error) {
	page, pageSize, offset := p.normalize()

	q := r.db.Model(&models.VPNUser{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []models.VPNUser
	err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).
		Preload("Devices").Preload("AccessKeys").
		Find(&users).Error
	_ = page
	return users, total, err
}

func (r *VPNUserRepository) FindByID(id uuid.UUID) (*models.VPNUser, error) {
	var user models.VPNUser
	err := r.db.Preload("Devices").Preload("AccessKeys").Preload("AccessKeys.Server").
		First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *VPNUserRepository) Create(user *models.VPNUser) error {
	return r.db.Create(user).Error
}

func (r *VPNUserRepository) Update(user *models.VPNUser) error {
	return r.db.Save(user).Error
}

func (r *VPNUserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.VPNUser{}, "id = ?", id).Error
}

func (r *VPNUserRepository) CountByStatus(status models.VPNUserStatus) (int64, error) {
	var count int64
	err := r.db.Model(&models.VPNUser{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *VPNUserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.VPNUser{}).Count(&count).Error
	return count, err
}

// FindExpired returns active users whose expiry has passed, for the
// expiration sweep that revokes VPN access.
func (r *VPNUserRepository) FindExpired() ([]models.VPNUser, error) {
	var users []models.VPNUser
	err := r.db.Where("status = ? AND expires_at IS NOT NULL AND expires_at < now()", models.VPNUserStatusActive).
		Find(&users).Error
	return users, err
}

func (r *VPNUserRepository) SumTrafficUsed() (int64, error) {
	var total int64
	err := r.db.Model(&models.VPNUser{}).Select("COALESCE(SUM(traffic_used_bytes), 0)").Scan(&total).Error
	return total, err
}
