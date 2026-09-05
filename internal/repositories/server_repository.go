package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

type ServerRepository struct {
	db *gorm.DB
}

func (r *ServerRepository) List(p Pagination) ([]models.VPNServer, int64, error) {
	page, pageSize, offset := p.normalize()

	var total int64
	if err := r.db.Model(&models.VPNServer{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var servers []models.VPNServer
	err := r.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&servers).Error
	_ = page
	return servers, total, err
}

func (r *ServerRepository) All() ([]models.VPNServer, error) {
	var servers []models.VPNServer
	err := r.db.Find(&servers).Error
	return servers, err
}

func (r *ServerRepository) FindByID(id uuid.UUID) (*models.VPNServer, error) {
	var server models.VPNServer
	if err := r.db.First(&server, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

func (r *ServerRepository) Create(server *models.VPNServer) error {
	return r.db.Create(server).Error
}

func (r *ServerRepository) Update(server *models.VPNServer) error {
	return r.db.Save(server).Error
}

func (r *ServerRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.VPNServer{}, "id = ?", id).Error
}
