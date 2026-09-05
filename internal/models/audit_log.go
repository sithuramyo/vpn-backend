package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/datatypes"
)

type AuditAction string

const (
	ActionAdminLogin         AuditAction = "ADMIN_LOGIN"
	ActionAdminDisabled      AuditAction = "ADMIN_DISABLED"
	ActionUserCreated        AuditAction = "USER_CREATED"
	ActionUserUpdated        AuditAction = "USER_UPDATED"
	ActionUserDisabled       AuditAction = "USER_DISABLED"
	ActionDeviceRevoked      AuditAction = "DEVICE_REVOKED"
	ActionAccessKeyCreated   AuditAction = "ACCESS_KEY_CREATED"
	ActionAccessKeyRevoked   AuditAction = "ACCESS_KEY_REVOKED"
	ActionAccessKeyRotated   AuditAction = "ACCESS_KEY_ROTATED"
	ActionServerCreated      AuditAction = "SERVER_CREATED"
	ActionServerUpdated      AuditAction = "SERVER_UPDATED"
	ActionServerDeleted      AuditAction = "SERVER_DELETED"
)

type AuditLog struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID      *uuid.UUID     `gorm:"column:admin_id;type:uuid;index" json:"admin_id,omitempty"`
	Action       AuditAction    `gorm:"not null;index" json:"action"`
	ResourceType string         `gorm:"column:resource_type" json:"resource_type"`
	ResourceID   *uuid.UUID     `gorm:"column:resource_id;type:uuid" json:"resource_id,omitempty"`
	Metadata     datatypes.JSON `json:"metadata,omitempty"`
	IPAddress    string         `gorm:"column:ip_address" json:"ip_address"`
	CreatedAt    time.Time      `json:"created_at"`

	Admin *Admin `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
