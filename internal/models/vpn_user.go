package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VPNUserStatus string

const (
	VPNUserStatusActive   VPNUserStatus = "ACTIVE"
	VPNUserStatusDisabled VPNUserStatus = "DISABLED"
	VPNUserStatusExpired  VPNUserStatus = "EXPIRED"
)

type VPNUser struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email             string         `json:"email"`
	Name              string         `json:"name"`
	Status            VPNUserStatus  `gorm:"not null" json:"status"`
	ExpiresAt         *time.Time     `gorm:"column:expires_at" json:"expires_at,omitempty"`
	TrafficLimitBytes int64          `gorm:"column:traffic_limit_bytes" json:"traffic_limit_bytes"`
	TrafficUsedBytes  int64          `gorm:"column:traffic_used_bytes" json:"traffic_used_bytes"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	Devices     []VPNDevice  `gorm:"foreignKey:VPNUserID" json:"devices,omitempty"`
	AccessKeys  []AccessKey  `gorm:"foreignKey:VPNUserID" json:"access_keys,omitempty"`
}

func (VPNUser) TableName() string { return "vpn_users" }

func (u *VPNUser) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u *VPNUser) IsExpired() bool {
	return u.ExpiresAt != nil && u.ExpiresAt.Before(time.Now())
}

func (u *VPNUser) IsOverQuota() bool {
	return u.TrafficLimitBytes > 0 && u.TrafficUsedBytes >= u.TrafficLimitBytes
}
