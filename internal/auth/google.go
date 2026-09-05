// Package auth verifies Google-issued identity tokens and issues the
// backend's own session tokens. The frontend performs the Google OAuth
// dance (via Auth.js) and hands this backend the raw Google ID token; this
// package is the only place that trusts Google's signature, and it never
// trusts a role or status claimed by the client.
package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const googleCertsURL = "https://www.googleapis.com/oauth2/v3/certs"

var validIssuers = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

type GoogleIdentity struct {
	Sub     string
	Email   string
	Name    string
	Picture string
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

// GoogleVerifier fetches and caches Google's public signing keys to verify
// ID token signatures without a round trip per request.
type GoogleVerifier struct {
	clientID string
	httpc    *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{
		clientID: clientID,
		httpc:    &http.Client{Timeout: 5 * time.Second},
		keys:     make(map[string]*rsa.PublicKey),
	}
}

func (v *GoogleVerifier) Verify(idToken string) (*GoogleIdentity, error) {
	if v.clientID == "" {
		return nil, fmt.Errorf("google client id not configured")
	}

	token, err := jwt.Parse(idToken, v.keyFunc, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, fmt.Errorf("invalid google id token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid google id token claims")
	}

	iss, _ := claims["iss"].(string)
	if !validIssuers[iss] {
		return nil, fmt.Errorf("unexpected issuer: %s", iss)
	}

	aud, _ := claims["aud"].(string)
	if aud != v.clientID {
		return nil, fmt.Errorf("token audience does not match configured client id")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("token missing sub claim")
	}

	emailVerified, _ := claims["email_verified"].(bool)
	if !emailVerified {
		return nil, fmt.Errorf("google email is not verified")
	}

	identity := &GoogleIdentity{
		Sub:     sub,
		Email:   asString(claims["email"]),
		Name:    asString(claims["name"]),
		Picture: asString(claims["picture"]),
	}
	return identity, nil
}

func (v *GoogleVerifier) keyFunc(token *jwt.Token) (any, error) {
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("token missing kid header")
	}

	key := v.cachedKey(kid)
	if key != nil {
		return key, nil
	}

	if err := v.refreshKeys(); err != nil {
		return nil, err
	}

	key = v.cachedKey(kid)
	if key == nil {
		return nil, fmt.Errorf("unknown signing key: %s", kid)
	}
	return key, nil
}

func (v *GoogleVerifier) cachedKey(kid string) *rsa.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if time.Now().After(v.expiresAt) {
		return nil
	}
	return v.keys[kid]
}

func (v *GoogleVerifier) refreshKeys() error {
	resp, err := v.httpc.Get(googleCertsURL)
	if err != nil {
		return fmt.Errorf("fetch google certs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read google certs: %w", err)
	}

	var set jwkSet
	if err := json.Unmarshal(body, &set); err != nil {
		return fmt.Errorf("parse google certs: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := jwkToRSAPublicKey(k)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}

	cacheDuration := 1 * time.Hour
	if cc := resp.Header.Get("Cache-Control"); cc != "" {
		if d := parseMaxAge(cc); d > 0 {
			cacheDuration = d
		}
	}

	v.mu.Lock()
	v.keys = keys
	v.expiresAt = time.Now().Add(cacheDuration)
	v.mu.Unlock()

	return nil
}

func jwkToRSAPublicKey(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

func parseMaxAge(cacheControl string) time.Duration {
	var maxAge int
	_, err := fmt.Sscanf(cacheControl, "max-age=%d", &maxAge)
	if err != nil || maxAge <= 0 {
		return 0
	}
	return time.Duration(maxAge) * time.Second
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
