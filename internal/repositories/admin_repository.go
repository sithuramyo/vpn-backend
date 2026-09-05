package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

type AdminRepository struct {
	db *gorm.DB
}

func (r *AdminRepository) FindByGoogleSub(sub string) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.Where("google_sub = ?", sub).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *AdminRepository) FindByID(id uuid.UUID) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.First(&admin, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *AdminRepository) List(p Pagination) ([]models.Admin, int64, error) {
	page, pageSize, offset := p.normalize()

	var total int64
	if err := r.db.Model(&models.Admin{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var admins []models.Admin
	err := r.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	_ = page
	return admins, total, err
}

func (r *AdminRepository) Update(admin *models.Admin) error {
	return r.db.Save(admin).Error
}

func (r *AdminRepository) Create(admin *models.Admin) error {
	return r.db.Create(admin).Error
}

func (r *AdminRepository) TouchLastLogin(id uuid.UUID) error {
	return r.db.Model(&models.Admin{}).Where("id = ?", id).Update("last_login_at", gorm.Expr("now()")).Error
}
