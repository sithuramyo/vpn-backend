package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccessKeyStatus string

const (
	AccessKeyStatusActive  AccessKeyStatus = "ACTIVE"
	AccessKeyStatusRevoked AccessKeyStatus = "REVOKED"
	AccessKeyStatusExpired AccessKeyStatus = "EXPIRED"
)

type AccessKeyProtocol string

const (
	ProtocolShadowsocks AccessKeyProtocol = "SHADOWSOCKS"
)

// AccessKey is the control-plane record of a VPN credential. The actual
// Shadowsocks secret lives with the VPN provider (see internal/vpn); this
// table only stores a reference used to look it up on demand.
type AccessKey struct {
	ID                uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	VPNUserID         uuid.UUID         `gorm:"column:vpn_user_id;type:uuid;not null;index" json:"vpn_user_id"`
	VPNServerID       uuid.UUID         `gorm:"column:vpn_server_id;type:uuid;not null;index" json:"vpn_server_id"`
	Name              string            `json:"name"`
	SecretReference   string            `gorm:"column:secret_reference;not null" json:"-"`
	EncryptedSecret   string            `gorm:"column:encrypted_secret" json:"-"`
	Port              int               `json:"-"`
	Cipher            string            `json:"cipher"`
	Protocol          AccessKeyProtocol `gorm:"not null" json:"protocol"`
	TCPEnabled        bool              `gorm:"column:tcp_enabled" json:"tcp_enabled"`
	UDPEnabled        bool              `gorm:"column:udp_enabled" json:"udp_enabled"`
	WebSocketEnabled  bool              `gorm:"column:websocket_enabled" json:"websocket_enabled"`
	WebSocketPath     string            `gorm:"column:websocket_path" json:"websocket_path,omitempty"`
	WebSocketUDPPath  string            `gorm:"column:websocket_udp_path" json:"websocket_udp_path,omitempty"`
	ExpiresAt         *time.Time        `gorm:"column:expires_at" json:"expires_at,omitempty"`
	TrafficLimitBytes int64             `gorm:"column:traffic_limit_bytes" json:"traffic_limit_bytes"`
	TrafficUsedBytes  int64             `gorm:"column:traffic_used_bytes" json:"traffic_used_bytes"`
	Status            AccessKeyStatus   `gorm:"not null" json:"status"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`

	User   *VPNUser   `gorm:"foreignKey:VPNUserID" json:"user,omitempty"`
	Server *VPNServer `gorm:"foreignKey:VPNServerID" json:"server,omitempty"`
}

func (AccessKey) TableName() string { return "access_keys" }

func (k *AccessKey) BeforeCreate(tx *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}

func (k *AccessKey) IsExpired() bool {
	return k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now())
}
