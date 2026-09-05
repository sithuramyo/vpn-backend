package services_test

import (
	"context"
	"testing"

	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
	"vpn-backend/internal/testutil"
)

func TestUserServiceCreateAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repos := repositories.New(db)
	provider := testutil.NewFakeVPNProvider()
	svc := services.NewUserService(repos.Users, repos.Devices, repos.AccessKeys, provider)

	user, err := svc.Create(services.CreateUserInput{Name: "Alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if user.Status != models.VPNUserStatusActive {
		t.Fatalf("expected new user to be ACTIVE, got %s", user.Status)
	}

	got, err := svc.Get(user.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("expected email to round-trip, got %q", got.Email)
	}
}

func TestUserServiceDisableRevokesAccessKeysAndDevices(t *testing.T) {
	db := testutil.NewTestDB(t)
	repos := repositories.New(db)
	provider := testutil.NewFakeVPNProvider()
	users := services.NewUserService(repos.Users, repos.Devices, repos.AccessKeys, provider)

	user, err := users.Create(services.CreateUserInput{Name: "Bob", Email: "bob@example.com"})
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	server := &models.VPNServer{Name: "sg-1", Hostname: "vpn.thestrm.space", Status: models.ServerStatusOnline}
	if err := repos.Servers.Create(server); err != nil {
		t.Fatalf("create server failed: %v", err)
	}

	key := &models.AccessKey{
		VPNUserID: user.ID, VPNServerID: server.ID, Name: "bob-key",
		SecretReference: "key-1", EncryptedSecret: "unused", Port: 17508,
		Protocol: models.ProtocolShadowsocks, Status: models.AccessKeyStatusActive,
	}
	if err := repos.AccessKeys.Create(key); err != nil {
		t.Fatalf("create access key failed: %v", err)
	}

	device := &models.VPNDevice{VPNUserID: user.ID, Name: "bob-phone", Platform: models.PlatformAndroid, Status: models.DeviceStatusActive}
	if err := repos.Devices.Create(device); err != nil {
		t.Fatalf("create device failed: %v", err)
	}

	if _, err := users.Disable(context.Background(), user.ID); err != nil {
		t.Fatalf("disable failed: %v", err)
	}

	updatedKey, err := repos.AccessKeys.FindByID(key.ID)
	if err != nil {
		t.Fatalf("find key failed: %v", err)
	}
	if updatedKey.Status != models.AccessKeyStatusRevoked {
		t.Fatalf("expected access key to be revoked, got %s", updatedKey.Status)
	}
	if len(provider.Revoked) != 1 || provider.Revoked[0] != "key-1" {
		t.Fatalf("expected provider.RevokeAccessKey to be called with key-1, got %v", provider.Revoked)
	}

	updatedDevice, err := repos.Devices.FindByID(device.ID)
	if err != nil {
		t.Fatalf("find device failed: %v", err)
	}
	if updatedDevice.Status != models.DeviceStatusRevoked {
		t.Fatalf("expected device to be revoked, got %s", updatedDevice.Status)
	}

	updatedUser, err := users.Get(user.ID)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if updatedUser.Status != models.VPNUserStatusDisabled {
		t.Fatalf("expected user to be DISABLED, got %s", updatedUser.Status)
	}
}
