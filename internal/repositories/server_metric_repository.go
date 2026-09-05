package repositories

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

type ServerMetricRepository struct {
	db *gorm.DB
}

func (r *ServerMetricRepository) Create(m *models.ServerMetric) error {
	return r.db.Create(m).Error
}

func (r *ServerMetricRepository) Latest(serverID uuid.UUID) (*models.ServerMetric, error) {
	var m models.ServerMetric
	err := r.db.Where("vpn_server_id = ?", serverID).Order("recorded_at DESC").First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ServerMetricRepository) Since(serverID uuid.UUID, since time.Time) ([]models.ServerMetric, error) {
	var metrics []models.ServerMetric
	err := r.db.Where("vpn_server_id = ? AND recorded_at >= ?", serverID, since).
		Order("recorded_at ASC").
		Find(&metrics).Error
	return metrics, err
}
