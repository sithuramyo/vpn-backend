package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DevicePlatform string

const (
	PlatformAndroid DevicePlatform = "ANDROID"
	PlatformIOS     DevicePlatform = "IOS"
	PlatformWindows DevicePlatform = "WINDOWS"
	PlatformMacOS   DevicePlatform = "MACOS"
)

type DeviceStatus string

const (
	DeviceStatusActive   DeviceStatus = "ACTIVE"
	DeviceStatusDisabled DeviceStatus = "DISABLED"
	DeviceStatusRevoked  DeviceStatus = "REVOKED"
)

type VPNDevice struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	VPNUserID  uuid.UUID      `gorm:"column:vpn_user_id;type:uuid;not null;index" json:"vpn_user_id"`
	Name       string         `json:"name"`
	Platform   DevicePlatform `gorm:"not null" json:"platform"`
	Status     DeviceStatus   `gorm:"not null" json:"status"`
	LastSeenAt *time.Time     `gorm:"column:last_seen_at" json:"last_seen_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`

	User *VPNUser `gorm:"foreignKey:VPNUserID" json:"user,omitempty"`
}

func (VPNDevice) TableName() string { return "vpn_devices" }

func (d *VPNDevice) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
