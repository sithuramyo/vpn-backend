package vpn

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OutlineShadowsocksProvider talks to a Jigsaw Outline server's Management
// REST API (https://github.com/Jigsaw-Code/outline-server), a mature,
// widely deployed Shadowsocks implementation. This client only orchestrates
// that API — it never touches Shadowsocks framing, ciphers, or key
// derivation directly.
type OutlineShadowsocksProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewOutlineShadowsocksProvider builds a client for an Outline server's
// management API.
//
// apiURL is the full management URL Outline prints on setup, e.g.
// "https://203.0.113.5:41287/aBcDeFgH123". Outline's management API is
// served on a self-signed certificate, so instead of validating it against
// a CA (there is none), the caller pins the certificate's SHA-256
// fingerprint, exactly as the official Outline Manager app does. If
// certSHA256 is empty, the system CA pool is used instead (suitable when
// the API sits behind a properly-issued TLS certificate, e.g. via Caddy).
func NewOutlineShadowsocksProvider(apiURL, certSHA256 string) *OutlineShadowsocksProvider {
	client := &http.Client{Timeout: 10 * time.Second}

	if certSHA256 != "" {
		pinned := strings.ToLower(strings.ReplaceAll(certSHA256, ":", ""))
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // fingerprint pinning below replaces CA verification
				VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
					for _, raw := range rawCerts {
						sum := sha256.Sum256(raw)
						if hex.EncodeToString(sum[:]) == pinned {
							return nil
						}
					}
					return fmt.Errorf("outline management API certificate did not match pinned fingerprint")
				},
			},
		}
	}

	return &OutlineShadowsocksProvider{
		baseURL:    strings.TrimRight(apiURL, "/"),
		httpClient: client,
	}
}

type outlineAccessKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Password  string `json:"password"`
	Port      int    `json:"port"`
	Method    string `json:"method"`
	AccessURL string `json:"accessUrl"`
}

type outlineServerInfo struct {
	Name           string `json:"name"`
	ServerID       string `json:"serverId"`
	MetricsEnabled bool   `json:"metricsEnabled"`
	Version        string `json:"version"`
}

type outlineTransferMetrics struct {
	BytesTransferredByUserID map[string]int64 `json:"bytesTransferredByUserId"`
}

func (p *OutlineShadowsocksProvider) CreateAccessKey(ctx context.Context, spec AccessKeySpec) (*ProvisionedKey, error) {
	var created outlineAccessKey
	if err := p.doJSON(ctx, http.MethodPost, "/access-keys", nil, &created); err != nil {
		return nil, fmt.Errorf("outline: create access key: %w", err)
	}

	if spec.Name != "" {
		if err := p.renameAccessKey(ctx, created.ID, spec.Name); err != nil {
			return nil, err
		}
	}
	if spec.TrafficLimitBytes > 0 {
		if err := p.setDataLimit(ctx, created.ID, spec.TrafficLimitBytes); err != nil {
			return nil, err
		}
	}

	return &ProvisionedKey{
		ProviderKeyID: created.ID,
		Password:      created.Password,
		Method:        created.Method,
		Port:          created.Port,
	}, nil
}

func (p *OutlineShadowsocksProvider) RevokeAccessKey(ctx context.Context, providerKeyID string) error {
	path := fmt.Sprintf("/access-keys/%s", providerKeyID)
	if err := p.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("outline: revoke access key: %w", err)
	}
	return nil
}

func (p *OutlineShadowsocksProvider) RotateAccessKey(ctx context.Context, providerKeyID string, spec AccessKeySpec) (*ProvisionedKey, error) {
	// The Outline management API has no in-place "rotate secret" call, so
	// rotation is a revoke-then-create against the same server, giving the
	// key a brand-new password while its control-plane row keeps its own
	// identity.
	if err := p.RevokeAccessKey(ctx, providerKeyID); err != nil {
		return nil, err
	}
	return p.CreateAccessKey(ctx, spec)
}

func (p *OutlineShadowsocksProvider) GetServerStatus(ctx context.Context) (*ServerStatus, error) {
	var info outlineServerInfo
	if err := p.doJSON(ctx, http.MethodGet, "/server", nil, &info); err != nil {
		return &ServerStatus{Healthy: false}, fmt.Errorf("outline: get server status: %w", err)
	}

	return &ServerStatus{
		Healthy:        true,
		Version:        info.Version,
		MetricsEnabled: info.MetricsEnabled,
	}, nil
}

func (p *OutlineShadowsocksProvider) GetUsage(ctx context.Context, providerKeyID string) (*UsageStats, error) {
	var metrics outlineTransferMetrics
	if err := p.doJSON(ctx, http.MethodGet, "/metrics/transfer", nil, &metrics); err != nil {
		return nil, fmt.Errorf("outline: get usage: %w", err)
	}
	return &UsageStats{BytesTransferred: metrics.BytesTransferredByUserID[providerKeyID]}, nil
}

func (p *OutlineShadowsocksProvider) renameAccessKey(ctx context.Context, keyID, name string) error {
	path := fmt.Sprintf("/access-keys/%s/name", keyID)
	body := map[string]string{"name": name}
	if err := p.doJSON(ctx, http.MethodPut, path, body, nil); err != nil {
		return fmt.Errorf("outline: rename access key: %w", err)
	}
	return nil
}

func (p *OutlineShadowsocksProvider) setDataLimit(ctx context.Context, keyID string, bytesLimit int64) error {
	path := fmt.Sprintf("/access-keys/%s/data-limit", keyID)
	body := map[string]int64{"bytes": bytesLimit}
	if err := p.doJSON(ctx, http.MethodPut, path, body, nil); err != nil {
		return fmt.Errorf("outline: set data limit: %w", err)
	}
	return nil
}

func (p *OutlineShadowsocksProvider) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
