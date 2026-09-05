package repositories

import (
	"time"

	"gorm.io/gorm"

	"vpn-backend/internal/models"
)

type AuditLogRepository struct {
	db *gorm.DB
}

type AuditLogFilter struct {
	AdminID      string
	Action       string
	ResourceType string
	ResourceID   string
	From         *time.Time
	To           *time.Time
}

func (r *AuditLogRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *AuditLogRepository) List(f AuditLogFilter, p Pagination) ([]models.AuditLog, int64, error) {
	page, pageSize, offset := p.normalize()

	q := r.db.Model(&models.AuditLog{})
	if f.AdminID != "" {
		q = q.Where("admin_id = ?", f.AdminID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.ResourceType != "" {
		q = q.Where("resource_type = ?", f.ResourceType)
	}
	if f.ResourceID != "" {
		q = q.Where("resource_id = ?", f.ResourceID)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []models.AuditLog
	err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Preload("Admin").Find(&logs).Error
	_ = page
	return logs, total, err
}
