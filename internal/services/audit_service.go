package services

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
)

type AuditService struct {
	repo *repositories.AuditLogRepository
}

func NewAuditService(repo *repositories.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Record persists an audit entry. Metadata must never contain raw secrets
// (access-key passwords, JWT/session tokens, OAuth secrets) — callers pass
// only identifying, non-sensitive fields.
func (s *AuditService) Record(adminID *uuid.UUID, action models.AuditAction, resourceType string, resourceID *uuid.UUID, ipAddress string, metadata map[string]any) error {
	var metaJSON datatypes.JSON
	if metadata != nil {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		metaJSON = encoded
	}

	entry := &models.AuditLog{
		AdminID:      adminID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metaJSON,
		IPAddress:    ipAddress,
	}
	return s.repo.Create(entry)
}

func (s *AuditService) List(f repositories.AuditLogFilter, p repositories.Pagination) ([]models.AuditLog, int64, error) {
	return s.repo.List(f, p)
}
