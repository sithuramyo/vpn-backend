package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminRole string

const (
	AdminRoleAdmin    AdminRole = "ADMIN"
	AdminRoleOperator AdminRole = "OPERATOR"
	AdminRoleViewer   AdminRole = "VIEWER"
)

type AdminStatus string

const (
	AdminStatusActive   AdminStatus = "ACTIVE"
	AdminStatusDisabled AdminStatus = "DISABLED"
)

type Admin struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	GoogleSub   string      `gorm:"column:google_sub;uniqueIndex;not null" json:"-"`
	Email       string      `gorm:"not null" json:"email"`
	Name        string      `json:"name"`
	PictureURL  string      `gorm:"column:picture_url" json:"picture_url"`
	Role        AdminRole   `gorm:"not null" json:"role"`
	Status      AdminStatus `gorm:"not null" json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	LastLoginAt *time.Time  `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
}

func (Admin) TableName() string { return "admins" }

func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (a *Admin) IsActive() bool { return a.Status == AdminStatusActive }
