package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// SessionManager issues and verifies the backend's own JWTs. These are
// deliberately minimal (subject + expiry only) — role and status are never
// embedded in the token, so a disabled admin or role change takes effect on
// the very next request instead of waiting for the token to expire.
type SessionManager struct {
	secret []byte
	expiry time.Duration
}

func NewSessionManager(secret string, expiry time.Duration) *SessionManager {
	return &SessionManager{secret: []byte(secret), expiry: expiry}
}

type sessionClaims struct {
	jwt.RegisteredClaims
}

func (s *SessionManager) Issue(adminID uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.expiry)
	claims := sessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign session token: %w", err)
	}
	return signed, expiresAt, nil
}

func (s *SessionManager) Verify(tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &sessionClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid session token: %w", err)
	}

	claims, ok := token.Claims.(*sessionClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid session claims")
	}

	adminID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid subject claim: %w", err)
	}

	return adminID, nil
}
