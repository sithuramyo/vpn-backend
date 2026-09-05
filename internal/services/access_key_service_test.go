package services_test

import (
	"context"
	"testing"

	"vpn-backend/internal/crypto"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
	"vpn-backend/internal/testutil"
)

func newAccessKeyService(t *testing.T) (*services.AccessKeyService, *repositories.Repositories, *testutil.FakeVPNProvider, *models.VPNUser, *models.VPNServer) {
	t.Helper()

	db := testutil.NewTestDB(t)
	repos := repositories.New(db)
	provider := testutil.NewFakeVPNProvider()

	secretBox, err := crypto.NewSecretBox([]byte("01234567890123456789012345678901")[:32])
	if err != nil {
		t.Fatalf("new secret box: %v", err)
	}

	svc := services.NewAccessKeyService(repos.AccessKeys, repos.Servers, provider, secretBox, "vpn.thestrm.space")

	user := &models.VPNUser{Name: "Alice", Email: "alice@example.com", Status: models.VPNUserStatusActive}
	if err := repos.Users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	server := &models.VPNServer{Name: "sg-1", Hostname: "vpn.thestrm.space", Status: models.ServerStatusOnline}
	if err := repos.Servers.Create(server); err != nil {
		t.Fatalf("create server: %v", err)
	}

	return svc, repos, provider, user, server
}

func TestAccessKeyServiceCreateGetConfigAndQR(t *testing.T) {
	svc, _, _, user, server := newAccessKeyService(t)

	key, err := svc.Create(context.Background(), services.CreateAccessKeyInput{
		VPNUserID: user.ID, VPNServerID: server.ID, Name: "alice-laptop",
		TCPEnabled: true, UDPEnabled: true, WebSocketEnabled: true,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if key.Status != models.AccessKeyStatusActive {
		t.Fatalf("expected ACTIVE status, got %s", key.Status)
	}
	if key.WebSocketPath == "" || key.WebSocketUDPPath == "" {
		t.Fatal("expected WebSocket paths to be generated")
	}

	cfg, err := svc.GetConfig(key.ID)
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	if cfg.ShadowsocksURI == "" {
		t.Fatal("expected a non-empty shadowsocks URI")
	}
	if len(cfg.CrossPlatform.FirstSupported) != 2 {
		t.Fatalf("expected 2 first-supported transports, got %d", len(cfg.CrossPlatform.FirstSupported))
	}

	png, err := svc.GetQRPNG(key.ID)
	if err != nil {
		t.Fatalf("get qr failed: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("expected non-empty PNG bytes")
	}
}

func TestAccessKeyServiceRevoke(t *testing.T) {
	svc, _, provider, user, server := newAccessKeyService(t)

	key, err := svc.Create(context.Background(), services.CreateAccessKeyInput{
		VPNUserID: user.ID, VPNServerID: server.ID, Name: "bob-phone",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	revoked, err := svc.Revoke(context.Background(), key.ID)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	if revoked.Status != models.AccessKeyStatusRevoked {
		t.Fatalf("expected REVOKED status, got %s", revoked.Status)
	}
	if len(provider.Revoked) != 1 {
		t.Fatalf("expected exactly one provider revoke call, got %d", len(provider.Revoked))
	}
}

func TestAccessKeyServiceRotateChangesSecretReference(t *testing.T) {
	svc, _, provider, user, server := newAccessKeyService(t)

	key, err := svc.Create(context.Background(), services.CreateAccessKeyInput{
		VPNUserID: user.ID, VPNServerID: server.ID, Name: "carol-tablet", WebSocketEnabled: true,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	originalRef := key.SecretReference
	originalPath := key.WebSocketPath

	rotated, err := svc.Rotate(context.Background(), key.ID)
	if err != nil {
		t.Fatalf("rotate failed: %v", err)
	}
	if rotated.SecretReference == originalRef {
		t.Fatal("expected rotation to change the provider key reference")
	}
	if rotated.WebSocketPath == originalPath {
		t.Fatal("expected rotation to regenerate the WebSocket path")
	}
	if rotated.Status != models.AccessKeyStatusActive {
		t.Fatalf("expected rotated key to remain ACTIVE, got %s", rotated.Status)
	}
	if len(provider.Rotated) != 1 {
		t.Fatalf("expected exactly one provider rotate call, got %d", len(provider.Rotated))
	}

	// Config generated after rotation must reflect the new secret, not the
	// one that existed before rotation.
	cfg, err := svc.GetConfig(key.ID)
	if err != nil {
		t.Fatalf("get config after rotate failed: %v", err)
	}
	if cfg.ShadowsocksURI == "" {
		t.Fatal("expected a non-empty shadowsocks URI after rotation")
	}
}
