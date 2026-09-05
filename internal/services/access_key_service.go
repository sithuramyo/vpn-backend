package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/crypto"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/vpn"
)

type AccessKeyService struct {
	keys      *repositories.AccessKeyRepository
	servers   *repositories.ServerRepository
	provider  vpn.VPNProvider
	secretBox *crypto.SecretBox
	vpnDomain string
}

func NewAccessKeyService(keys *repositories.AccessKeyRepository, servers *repositories.ServerRepository, provider vpn.VPNProvider, secretBox *crypto.SecretBox, vpnDomain string) *AccessKeyService {
	return &AccessKeyService{keys: keys, servers: servers, provider: provider, secretBox: secretBox, vpnDomain: vpnDomain}
}

type CreateAccessKeyInput struct {
	VPNUserID         uuid.UUID
	VPNServerID       uuid.UUID
	Name              string
	ExpiresAt         *time.Time
	TrafficLimitBytes int64
	TCPEnabled        bool
	UDPEnabled        bool
	WebSocketEnabled  bool
}

func (s *AccessKeyService) Create(ctx context.Context, input CreateAccessKeyInput) (*models.AccessKey, error) {
	server, err := s.servers.FindByID(input.VPNServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("server %w", ErrNotFound)
		}
		return nil, err
	}

	provisioned, err := s.provider.CreateAccessKey(ctx, vpn.AccessKeySpec{
		Name:              input.Name,
		TrafficLimitBytes: input.TrafficLimitBytes,
		Port:              server.TLSPort,
	})
	if err != nil {
		return nil, fmt.Errorf("provision access key: %w", err)
	}

	encryptedSecret, err := s.secretBox.Encrypt(provisioned.Password)
	if err != nil {
		return nil, err
	}

	key := &models.AccessKey{
		VPNUserID:         input.VPNUserID,
		VPNServerID:       input.VPNServerID,
		Name:              input.Name,
		SecretReference:   provisioned.ProviderKeyID,
		EncryptedSecret:   encryptedSecret,
		Port:              provisioned.Port,
		Cipher:            provisioned.Method,
		Protocol:          models.ProtocolShadowsocks,
		TCPEnabled:        input.TCPEnabled,
		UDPEnabled:        input.UDPEnabled,
		WebSocketEnabled:  input.WebSocketEnabled,
		ExpiresAt:         input.ExpiresAt,
		TrafficLimitBytes: input.TrafficLimitBytes,
		Status:            models.AccessKeyStatusActive,
	}

	if input.WebSocketEnabled {
		tcpPath, err := vpn.GenerateSecretPath()
		if err != nil {
			return nil, err
		}
		key.WebSocketPath = tcpPath

		if input.UDPEnabled {
			udpPath, err := vpn.GenerateSecretPath()
			if err != nil {
				return nil, err
			}
			key.WebSocketUDPPath = udpPath
		}
	}

	if err := s.keys.Create(key); err != nil {
		// Best-effort cleanup: don't leave an orphaned key on the data plane
		// if we failed to persist its control-plane record.
		_ = s.provider.RevokeAccessKey(ctx, provisioned.ProviderKeyID)
		return nil, err
	}

	key.Server = server
	return key, nil
}

func (s *AccessKeyService) List(f repositories.AccessKeyFilter, p repositories.Pagination) ([]models.AccessKey, int64, error) {
	return s.keys.List(f, p)
}

func (s *AccessKeyService) Get(id uuid.UUID) (*models.AccessKey, error) {
	key, err := s.keys.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("access key %w", ErrNotFound)
		}
		return nil, err
	}
	return key, nil
}

func (s *AccessKeyService) Revoke(ctx context.Context, id uuid.UUID) (*models.AccessKey, error) {
	key, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := s.provider.RevokeAccessKey(ctx, key.SecretReference); err != nil {
		return nil, fmt.Errorf("revoke on data plane: %w", err)
	}
	key.Status = models.AccessKeyStatusRevoked
	if err := s.keys.Update(key); err != nil {
		return nil, err
	}
	return key, nil
}

// Rotate replaces the underlying Shadowsocks credential (and, if
// WebSocket-enabled, its obfuscated path) while keeping the same
// control-plane access-key identity.
func (s *AccessKeyService) Rotate(ctx context.Context, id uuid.UUID) (*models.AccessKey, error) {
	key, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	provisioned, err := s.provider.RotateAccessKey(ctx, key.SecretReference, vpn.AccessKeySpec{
		Name:              key.Name,
		TrafficLimitBytes: key.TrafficLimitBytes,
		Port:              key.Server.TLSPort,
	})
	if err != nil {
		return nil, fmt.Errorf("rotate on data plane: %w", err)
	}

	encryptedSecret, err := s.secretBox.Encrypt(provisioned.Password)
	if err != nil {
		return nil, err
	}

	key.SecretReference = provisioned.ProviderKeyID
	key.EncryptedSecret = encryptedSecret
	key.Port = provisioned.Port
	key.Cipher = provisioned.Method
	key.Status = models.AccessKeyStatusActive

	if key.WebSocketEnabled {
		tcpPath, err := vpn.GenerateSecretPath()
		if err != nil {
			return nil, err
		}
		key.WebSocketPath = tcpPath
		if key.UDPEnabled {
			udpPath, err := vpn.GenerateSecretPath()
			if err != nil {
				return nil, err
			}
			key.WebSocketUDPPath = udpPath
		}
	}

	if err := s.keys.Update(key); err != nil {
		return nil, err
	}
	return key, nil
}

type AccessKeyConfig struct {
	ShadowsocksURI string                    `json:"shadowsocks_uri"`
	CrossPlatform  vpn.CrossPlatformConfig   `json:"cross_platform"`
}

// GetConfig decrypts the stored secret only for this one call, to hand an
// authorized administrator the client configuration on demand — it is
// never logged and never returned unless explicitly requested.
func (s *AccessKeyService) GetConfig(id uuid.UUID) (*AccessKeyConfig, error) {
	key, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	server, err := s.servers.FindByID(key.VPNServerID)
	if err != nil {
		return nil, err
	}

	password, err := s.secretBox.Decrypt(key.EncryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}

	// Clients connect to the server's public TLS port (443), not Outline's
	// internal data port (key.Port) - a TCP multiplexer in front of Caddy
	// (see scripts/sslh.conf.example) inspects each connection and routes
	// anything that isn't a real TLS handshake through to Outline. This
	// makes the VPN reachable on networks/carriers that block or filter
	// non-standard ports, since traffic on 443 is indistinguishable from
	// ordinary HTTPS at the network level.
	uri := vpn.ShadowsocksURI(key.Cipher, password, server.Hostname, server.TLSPort, key.Name)

	crossPlatform := vpn.BuildCrossPlatformConfig(vpn.ConfigParams{
		VPNDomain:        s.vpnDomain,
		Method:           key.Cipher,
		Password:         password,
		FallbackPort:     server.TLSPort,
		WebSocketPath:    key.WebSocketPath,
		WebSocketUDPPath: key.WebSocketUDPPath,
		SupportsUDP:      key.UDPEnabled,
	})

	return &AccessKeyConfig{ShadowsocksURI: uri, CrossPlatform: crossPlatform}, nil
}

func (s *AccessKeyService) GetQRPNG(id uuid.UUID) ([]byte, error) {
	cfg, err := s.GetConfig(id)
	if err != nil {
		return nil, err
	}
	return vpn.GenerateQRPNG(cfg.ShadowsocksURI, 256)
}

func (s *AccessKeyService) Delete(ctx context.Context, id uuid.UUID) error {
	key, err := s.Get(id)
	if err != nil {
		return err
	}
	if key.Status == models.AccessKeyStatusActive {
		if err := s.provider.RevokeAccessKey(ctx, key.SecretReference); err != nil {
			return fmt.Errorf("revoke on data plane: %w", err)
		}
	}
	return s.keys.Delete(id)
}

func (s *AccessKeyService) CountActive() (int64, error) {
	return s.keys.CountActive()
}

// SweepExpired revokes any access key whose expiry has passed.
func (s *AccessKeyService) SweepExpired(ctx context.Context) (int, error) {
	expired, err := s.keys.FindExpired()
	if err != nil {
		return 0, err
	}
	for i := range expired {
		key := &expired[i]
		if err := s.provider.RevokeAccessKey(ctx, key.SecretReference); err != nil {
			return 0, err
		}
		key.Status = models.AccessKeyStatusExpired
		if err := s.keys.Update(key); err != nil {
			return 0, err
		}
	}
	return len(expired), nil
}
