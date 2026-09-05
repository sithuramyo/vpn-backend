package vpn

import (
	"strings"
	"testing"
)

func TestGenerateSecretPathIsUniqueAndURLSafe(t *testing.T) {
	a, err := GenerateSecretPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := GenerateSecretPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == b {
		t.Fatalf("expected two distinct random paths, got %q twice", a)
	}
	if strings.ContainsAny(a, "+/=") {
		t.Fatalf("expected URL-safe base64 without padding, got %q", a)
	}
}

func TestShadowsocksURI(t *testing.T) {
	uri := ShadowsocksURI("chacha20-ietf-poly1305", "hunter2", "vpn.thestrm.space", 17508, "alice")

	if !strings.HasPrefix(uri, "ss://") {
		t.Fatalf("expected ss:// scheme, got %q", uri)
	}
	if !strings.Contains(uri, "vpn.thestrm.space:17508") {
		t.Fatalf("expected host:port in URI, got %q", uri)
	}
	if !strings.HasSuffix(uri, "#alice") {
		t.Fatalf("expected name fragment, got %q", uri)
	}
	if strings.Contains(uri, "hunter2") {
		t.Fatalf("expected password to be base64-encoded, not plaintext: %q", uri)
	}
}

func TestBuildCrossPlatformConfigWSSFirstWithFallback(t *testing.T) {
	cfg := BuildCrossPlatformConfig(ConfigParams{
		VPNDomain:        "vpn.thestrm.space",
		Method:           "chacha20-ietf-poly1305",
		Password:         "hunter2",
		FallbackPort:     17508,
		WebSocketPath:    "abc123",
		WebSocketUDPPath: "def456",
		SupportsUDP:      true,
	})

	if len(cfg.FirstSupported) != 2 {
		t.Fatalf("expected 2 first-supported options, got %d", len(cfg.FirstSupported))
	}
	if cfg.FirstSupported[0].Transport != "wss" {
		t.Fatalf("expected WSS to be tried first, got %q", cfg.FirstSupported[0].Transport)
	}
	if cfg.FirstSupported[1].Transport != "shadowsocks" || cfg.FirstSupported[1].Port != 17508 {
		t.Fatalf("expected shadowsocks fallback with fallback port, got %+v", cfg.FirstSupported[1])
	}
	if cfg.UDP == nil || !strings.HasSuffix(cfg.UDP.URL, "/def456/udp") {
		t.Fatalf("expected UDP option when SupportsUDP is true, got %+v", cfg.UDP)
	}
}

func TestBuildCrossPlatformConfigNoUDPWhenUnsupported(t *testing.T) {
	cfg := BuildCrossPlatformConfig(ConfigParams{
		VPNDomain:     "vpn.thestrm.space",
		Method:        "chacha20-ietf-poly1305",
		Password:      "hunter2",
		FallbackPort:  17508,
		WebSocketPath: "abc123",
		SupportsUDP:   false,
	})

	if cfg.UDP != nil {
		t.Fatalf("expected no UDP option when unsupported, got %+v", cfg.UDP)
	}
}
