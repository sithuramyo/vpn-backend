package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServerStatus string

const (
	ServerStatusOnline      ServerStatus = "ONLINE"
	ServerStatusOffline     ServerStatus = "OFFLINE"
	ServerStatusDegraded    ServerStatus = "DEGRADED"
	ServerStatusMaintenance ServerStatus = "MAINTENANCE"
)

type VPNServer struct {
	ID        uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string       `gorm:"not null" json:"name"`
	Hostname  string       `gorm:"not null" json:"hostname"`
	PublicIP  string       `gorm:"column:public_ip" json:"public_ip"`
	Country   string       `json:"country"`
	City      string       `json:"city"`
	Status    ServerStatus `gorm:"not null" json:"status"`
	VPNPort   int          `gorm:"column:vpn_port" json:"vpn_port"`
	TLSPort   int          `gorm:"column:tls_port" json:"tls_port"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (VPNServer) TableName() string { return "vpn_servers" }

func (s *VPNServer) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type ServerMetric struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	VPNServerID        uuid.UUID `gorm:"column:vpn_server_id;type:uuid;not null;index" json:"vpn_server_id"`
	CPUUsage           float64   `gorm:"column:cpu_usage" json:"cpu_usage"`
	MemoryUsage        float64   `gorm:"column:memory_usage" json:"memory_usage"`
	BandwidthIn        int64     `gorm:"column:bandwidth_in" json:"bandwidth_in"`
	BandwidthOut       int64     `gorm:"column:bandwidth_out" json:"bandwidth_out"`
	ActiveConnections  int       `gorm:"column:active_connections" json:"active_connections"`
	RecordedAt         time.Time `gorm:"column:recorded_at;autoCreateTime" json:"recorded_at"`
}

func (ServerMetric) TableName() string { return "server_metrics" }

func (m *ServerMetric) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
