package testutil

import (
	"context"
	"fmt"
	"sync"

	"vpn-backend/internal/vpn"
)

// FakeVPNProvider is an in-memory vpn.VPNProvider for tests, standing in
// for a real Outline server so service-layer logic can be exercised
// without network calls.
type FakeVPNProvider struct {
	mu       sync.Mutex
	nextID   int
	keys     map[string]bool
	Revoked  []string
	Rotated  []string
}

func NewFakeVPNProvider() *FakeVPNProvider {
	return &FakeVPNProvider{keys: make(map[string]bool)}
}

func (p *FakeVPNProvider) CreateAccessKey(_ context.Context, spec vpn.AccessKeySpec) (*vpn.ProvisionedKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nextID++
	id := fmt.Sprintf("key-%d", p.nextID)
	p.keys[id] = true
	return &vpn.ProvisionedKey{
		ProviderKeyID: id,
		Password:      "test-password-" + id,
		Method:        "chacha20-ietf-poly1305",
		Port:          17508,
	}, nil
}

func (p *FakeVPNProvider) RevokeAccessKey(_ context.Context, providerKeyID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.keys, providerKeyID)
	p.Revoked = append(p.Revoked, providerKeyID)
	return nil
}

func (p *FakeVPNProvider) RotateAccessKey(ctx context.Context, providerKeyID string, spec vpn.AccessKeySpec) (*vpn.ProvisionedKey, error) {
	p.mu.Lock()
	p.Rotated = append(p.Rotated, providerKeyID)
	p.mu.Unlock()

	if err := p.RevokeAccessKey(ctx, providerKeyID); err != nil {
		return nil, err
	}
	return p.CreateAccessKey(ctx, spec)
}

func (p *FakeVPNProvider) GetServerStatus(_ context.Context) (*vpn.ServerStatus, error) {
	return &vpn.ServerStatus{Healthy: true, Version: "test"}, nil
}

func (p *FakeVPNProvider) GetUsage(_ context.Context, providerKeyID string) (*vpn.UsageStats, error) {
	return &vpn.UsageStats{BytesTransferred: 0}, nil
}
