// Package vpn abstracts the data-plane VPN server behind a small interface.
// The control plane (this backend) never implements Shadowsocks, TLS, or
// WebSocket transport itself — it only talks to a mature, already-deployed
// implementation (Outline/Shadowsocks) over that implementation's own
// management API.
package vpn

import "context"

type AccessKeySpec struct {
	Name              string
	TrafficLimitBytes int64
	// Port pins the key to a specific data-plane port (e.g. the server's
	// public TLS port, so it's reachable on networks that block
	// non-standard ports). Outline's create/rotate calls otherwise just
	// attach the new key to whatever listener already exists rather than
	// consulting its own server-wide "port for new access keys" default,
	// so this must be set explicitly - zero leaves the choice to Outline.
	Port int
}

// ProvisionedKey is what the data plane hands back after creating a key:
// the credentials needed to build a client configuration, plus the
// provider's own identifier for later revoke/rotate/usage calls.
type ProvisionedKey struct {
	ProviderKeyID string
	Password      string
	Method        string
	Port          int
}

type UsageStats struct {
	BytesTransferred int64
}

type ServerStatus struct {
	Healthy            bool
	Version            string
	MetricsEnabled     bool
	ActiveConnections  int
}

// VPNProvider is implemented by whatever manages the actual Shadowsocks
// server (Outline today). Do not implement the VPN protocol or its
// cryptography here — only orchestrate an existing, mature implementation.
type VPNProvider interface {
	CreateAccessKey(ctx context.Context, spec AccessKeySpec) (*ProvisionedKey, error)
	RevokeAccessKey(ctx context.Context, providerKeyID string) error
	RotateAccessKey(ctx context.Context, providerKeyID string, spec AccessKeySpec) (*ProvisionedKey, error)
	GetServerStatus(ctx context.Context) (*ServerStatus, error)
	GetUsage(ctx context.Context, providerKeyID string) (*UsageStats, error)
}
