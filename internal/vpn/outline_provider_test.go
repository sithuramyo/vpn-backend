package vpn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFakeOutlineServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/access-keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s on /access-keys", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "1", "name": "", "password": "swordfish",
			"port": 17508, "method": "chacha20-ietf-poly1305",
			"accessUrl": "ss://example",
		})
	})
	mux.HandleFunc("/access-keys/1/name", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method %s on rename", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/access-keys/1/data-limit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/access-keys/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method %s on delete", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "sg-1", "serverId": "abc", "metricsEnabled": true, "version": "1.9.0",
		})
	})

	return httptest.NewServer(mux)
}

func TestOutlineShadowsocksProviderCreateAndRevoke(t *testing.T) {
	server := newFakeOutlineServer(t)
	defer server.Close()

	provider := NewOutlineShadowsocksProvider(server.URL, "")

	key, err := provider.CreateAccessKey(context.Background(), AccessKeySpec{Name: "alice", TrafficLimitBytes: 1000})
	if err != nil {
		t.Fatalf("unexpected error creating key: %v", err)
	}
	if key.ProviderKeyID != "1" || key.Password != "swordfish" || key.Port != 17508 {
		t.Fatalf("unexpected provisioned key: %+v", key)
	}

	if err := provider.RevokeAccessKey(context.Background(), key.ProviderKeyID); err != nil {
		t.Fatalf("unexpected error revoking key: %v", err)
	}
}

func TestOutlineShadowsocksProviderGetServerStatus(t *testing.T) {
	server := newFakeOutlineServer(t)
	defer server.Close()

	provider := NewOutlineShadowsocksProvider(server.URL, "")

	status, err := provider.GetServerStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Healthy || status.Version != "1.9.0" || !status.MetricsEnabled {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestOutlineShadowsocksProviderPropagatesErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider := NewOutlineShadowsocksProvider(server.URL, "")
	if _, err := provider.GetServerStatus(context.Background()); err == nil {
		t.Fatal("expected an error from a failing management API")
	}
}
