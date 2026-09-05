package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/vpn"
)

type ServerService struct {
	servers    *repositories.ServerRepository
	metrics    *repositories.ServerMetricRepository
	accessKeys *repositories.AccessKeyRepository
	provider   vpn.VPNProvider
}

func NewServerService(servers *repositories.ServerRepository, metrics *repositories.ServerMetricRepository, accessKeys *repositories.AccessKeyRepository, provider vpn.VPNProvider) *ServerService {
	return &ServerService{servers: servers, metrics: metrics, accessKeys: accessKeys, provider: provider}
}

type CreateServerInput struct {
	Name     string
	Hostname string
	PublicIP string
	Country  string
	City     string
	VPNPort  int
	TLSPort  int
}

func (s *ServerService) Create(input CreateServerInput) (*models.VPNServer, error) {
	server := &models.VPNServer{
		Name:     input.Name,
		Hostname: input.Hostname,
		PublicIP: input.PublicIP,
		Country:  input.Country,
		City:     input.City,
		Status:   models.ServerStatusOnline,
		VPNPort:  input.VPNPort,
		TLSPort:  input.TLSPort,
	}
	if err := s.servers.Create(server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *ServerService) List(p repositories.Pagination) ([]models.VPNServer, int64, error) {
	return s.servers.List(p)
}

func (s *ServerService) Get(id uuid.UUID) (*models.VPNServer, error) {
	server, err := s.servers.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("server %w", ErrNotFound)
		}
		return nil, err
	}
	return server, nil
}

type UpdateServerInput struct {
	Name    *string
	Status  *models.ServerStatus
	VPNPort *int
	TLSPort *int
}

func (s *ServerService) Update(id uuid.UUID, input UpdateServerInput) (*models.VPNServer, error) {
	server, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		server.Name = *input.Name
	}
	if input.Status != nil {
		server.Status = *input.Status
	}
	if input.VPNPort != nil {
		server.VPNPort = *input.VPNPort
	}
	if input.TLSPort != nil {
		server.TLSPort = *input.TLSPort
	}
	if err := s.servers.Update(server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *ServerService) Delete(id uuid.UUID) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.servers.Delete(id)
}

type HealthReport struct {
	Status             string  `json:"status"`
	CPUUsage           float64 `json:"cpu_usage"`
	MemoryUsage        float64 `json:"memory_usage"`
	UptimeSeconds      float64 `json:"uptime_seconds"`
	ActiveConnections  int     `json:"active_connections"`
	ShadowsocksHealthy bool    `json:"shadowsocks_healthy"`
	CaddyHealthy       bool    `json:"caddy_healthy"`
}

// Health combines the data-plane provider's own status call with
// host-level system metrics. It never proxies or inspects VPN traffic
// itself — only orchestrates existing health signals.
func (s *ServerService) Health(ctx context.Context, id uuid.UUID) (*HealthReport, error) {
	if _, err := s.Get(id); err != nil {
		return nil, err
	}

	providerStatus, providerErr := s.provider.GetServerStatus(ctx)
	sysMetrics, sysErr := vpn.ReadSystemMetrics()
	if sysErr != nil {
		sysMetrics = &vpn.SystemMetrics{}
	}

	report := &HealthReport{
		CPUUsage:           sysMetrics.CPUUsagePercent,
		MemoryUsage:        sysMetrics.MemoryUsedPct,
		UptimeSeconds:      sysMetrics.UptimeSeconds,
		ShadowsocksHealthy: providerErr == nil && providerStatus != nil && providerStatus.Healthy,
		CaddyHealthy:       vpn.IsServiceActive("caddy"),
	}

	if report.ShadowsocksHealthy {
		report.Status = "healthy"
	} else {
		report.Status = "degraded"
	}

	_ = s.metrics.Create(&models.ServerMetric{
		VPNServerID:       id,
		CPUUsage:          report.CPUUsage,
		MemoryUsage:       report.MemoryUsage,
		ActiveConnections: report.ActiveConnections,
	})

	return report, nil
}

// MetricsHistory returns the recorded health samples for a server within
// the given window, for the CPU/memory/bandwidth/connections charts on the
// server detail page.
func (s *ServerService) MetricsHistory(id uuid.UUID, window time.Duration) ([]models.ServerMetric, error) {
	if _, err := s.Get(id); err != nil {
		return nil, err
	}
	return s.metrics.Since(id, time.Now().Add(-window))
}
