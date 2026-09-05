package vpn

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
)

// GenerateSecretPath returns a random, URL-safe path segment used as the
// unguessable WebSocket route for a single access key
// (wss://host/<path>/tcp). It contains no VPN secret material itself, but
// must still be treated as sensitive since it is part of how a client
// reaches its Shadowsocks session.
func GenerateSecretPath() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate secret path: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// ShadowsocksURI builds a standard SIP002 ss:// URI
// (ss://base64(method:password)@host:port#name), the widely supported
// format every Shadowsocks-compatible client can import directly or scan
// as a QR code.
func ShadowsocksURI(method, password, host string, port int, name string) string {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).
		EncodeToString([]byte(fmt.Sprintf("%s:%s", method, password)))
	u := url.URL{
		Scheme:   "ss",
		User:     url.User(userinfo),
		Host:     fmt.Sprintf("%s:%d", host, port),
		Fragment: name,
	}
	return u.String()
}

// TransportOption is one entry in a "first-supported" transport list: the
// client tries each option in order and uses the first one it can
// establish, so a WSS/TLS route can be offered ahead of a plain
// Shadowsocks fallback without breaking older clients.
type TransportOption struct {
	Transport string `json:"transport"`

	// wss
	URL string `json:"url,omitempty"`

	// shadowsocks (used standalone, and nested under a wss option)
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Method   string `json:"method,omitempty"`
	Password string `json:"password,omitempty"`
}

type CrossPlatformConfig struct {
	FirstSupported []TransportOption `json:"first-supported"`
	UDP            *TransportOption  `json:"udp,omitempty"`
}

type ConfigParams struct {
	VPNDomain        string
	Method           string
	Password         string
	FallbackPort     int
	WebSocketPath    string
	WebSocketUDPPath string
	SupportsUDP      bool
}

// BuildCrossPlatformConfig implements the WSS-first, Shadowsocks-fallback
// model described for clients that understand a "first-supported"
// transport list. Callers must only advertise the WebSocket options to
// clients actually capable of using them (see doc: "do not claim a
// platform supports WSS unless the target client actually supports it").
func BuildCrossPlatformConfig(p ConfigParams) CrossPlatformConfig {
	options := []TransportOption{
		{
			Transport: "wss",
			URL:       fmt.Sprintf("wss://%s/%s/tcp", p.VPNDomain, p.WebSocketPath),
			Method:    p.Method,
			Password:  p.Password,
		},
		{
			Transport: "shadowsocks",
			Host:      p.VPNDomain,
			Port:      p.FallbackPort,
			Method:    p.Method,
			Password:  p.Password,
		},
	}

	cfg := CrossPlatformConfig{FirstSupported: options}

	if p.SupportsUDP && p.WebSocketUDPPath != "" {
		cfg.UDP = &TransportOption{
			Transport: "wss",
			URL:       fmt.Sprintf("wss://%s/%s/udp", p.VPNDomain, p.WebSocketUDPPath),
			Method:    p.Method,
			Password:  p.Password,
		}
	}

	return cfg
}
