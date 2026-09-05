package services

import (
	"time"

	"vpn-backend/internal/repositories"
)

type UsageService struct {
	users      *repositories.VPNUserRepository
	accessKeys *repositories.AccessKeyRepository
	servers    *repositories.ServerRepository
	metrics    *repositories.ServerMetricRepository
}

func NewUsageService(users *repositories.VPNUserRepository, accessKeys *repositories.AccessKeyRepository, servers *repositories.ServerRepository, metrics *repositories.ServerMetricRepository) *UsageService {
	return &UsageService{users: users, accessKeys: accessKeys, servers: servers, metrics: metrics}
}

type UsageSummary struct {
	TotalTrafficBytes int64                `json:"total_traffic_bytes"`
	DailyBandwidth    []BandwidthDataPoint `json:"daily_bandwidth"`
	MonthlyBandwidth  []BandwidthDataPoint `json:"monthly_bandwidth"`
}

type BandwidthDataPoint struct {
	Timestamp         time.Time `json:"timestamp"`
	BytesIn           int64     `json:"bytes_in"`
	BytesOut          int64     `json:"bytes_out"`
	ActiveConnections int       `json:"active_connections"`
}

func (s *UsageService) Summary() (*UsageSummary, error) {
	total, err := s.users.SumTrafficUsed()
	if err != nil {
		return nil, err
	}

	daily, err := s.bandwidthSince(24 * time.Hour)
	if err != nil {
		return nil, err
	}
	monthly, err := s.bandwidthSince(30 * 24 * time.Hour)
	if err != nil {
		return nil, err
	}

	return &UsageSummary{
		TotalTrafficBytes: total,
		DailyBandwidth:    daily,
		MonthlyBandwidth:  monthly,
	}, nil
}

func (s *UsageService) bandwidthSince(window time.Duration) ([]BandwidthDataPoint, error) {
	servers, err := s.servers.All()
	if err != nil {
		return nil, err
	}

	since := time.Now().Add(-window)
	var points []BandwidthDataPoint
	for _, server := range servers {
		samples, err := s.metrics.Since(server.ID, since)
		if err != nil {
			return nil, err
		}
		for _, sample := range samples {
			points = append(points, BandwidthDataPoint{
				Timestamp:         sample.RecordedAt,
				BytesIn:           sample.BandwidthIn,
				BytesOut:          sample.BandwidthOut,
				ActiveConnections: sample.ActiveConnections,
			})
		}
	}
	return points, nil
}

type UserUsage struct {
	UserID            string `json:"user_id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	TrafficUsedBytes  int64  `json:"traffic_used_bytes"`
	TrafficLimitBytes int64  `json:"traffic_limit_bytes"`
}

func (s *UsageService) ByUser(p repositories.Pagination) ([]UserUsage, int64, error) {
	users, total, err := s.users.List(repositories.UserFilter{}, p)
	if err != nil {
		return nil, 0, err
	}

	result := make([]UserUsage, 0, len(users))
	for _, u := range users {
		result = append(result, UserUsage{
			UserID:            u.ID.String(),
			Name:              u.Name,
			Email:             u.Email,
			TrafficUsedBytes:  u.TrafficUsedBytes,
			TrafficLimitBytes: u.TrafficLimitBytes,
		})
	}
	return result, total, nil
}

type ServerUsage struct {
	ServerID         string `json:"server_id"`
	Name             string `json:"name"`
	TrafficUsedBytes int64  `json:"traffic_used_bytes"`
}

func (s *UsageService) ByServer() ([]ServerUsage, error) {
	servers, err := s.servers.All()
	if err != nil {
		return nil, err
	}

	result := make([]ServerUsage, 0, len(servers))
	for _, server := range servers {
		total, err := s.accessKeys.SumTrafficByServer(server.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, ServerUsage{
			ServerID:         server.ID.String(),
			Name:             server.Name,
			TrafficUsedBytes: total,
		})
	}
	return result, nil
}
