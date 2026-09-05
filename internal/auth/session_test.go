package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSessionManagerIssueAndVerify(t *testing.T) {
	sm := NewSessionManager("test-secret", time.Hour)
	adminID := uuid.New()

	token, expiresAt, err := sm.Issue(adminID)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("expected expiry in the future")
	}

	gotID, err := sm.Verify(token)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if gotID != adminID {
		t.Fatalf("expected admin id %s, got %s", adminID, gotID)
	}
}

func TestSessionManagerRejectsExpiredToken(t *testing.T) {
	sm := NewSessionManager("test-secret", -time.Hour)
	token, _, err := sm.Issue(uuid.New())
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	if _, err := sm.Verify(token); err == nil {
		t.Fatal("expected an expired token to fail verification")
	}
}

func TestSessionManagerRejectsTokenFromDifferentSecret(t *testing.T) {
	sm1 := NewSessionManager("secret-one", time.Hour)
	sm2 := NewSessionManager("secret-two", time.Hour)

	token, _, err := sm1.Issue(uuid.New())
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	if _, err := sm2.Verify(token); err == nil {
		t.Fatal("expected verification with a different secret to fail")
	}
}
