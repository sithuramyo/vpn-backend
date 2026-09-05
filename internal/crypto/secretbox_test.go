package crypto

import "testing"

func testKey() []byte {
	return []byte("01234567890123456789012345678901")[:32]
}

func TestSecretBoxRoundTrip(t *testing.T) {
	box, err := NewSecretBox(testKey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ciphertext, err := box.Encrypt("hunter2")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if ciphertext == "hunter2" {
		t.Fatal("ciphertext must not equal plaintext")
	}

	plaintext, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if plaintext != "hunter2" {
		t.Fatalf("expected round-trip to recover plaintext, got %q", plaintext)
	}
}

func TestSecretBoxRejectsWrongKeySize(t *testing.T) {
	if _, err := NewSecretBox([]byte("too-short")); err == nil {
		t.Fatal("expected an error for a non-32-byte key")
	}
}

func TestSecretBoxDecryptFailsWithWrongKey(t *testing.T) {
	box1, _ := NewSecretBox(testKey())
	otherKey := []byte("abcdefghijklmnopqrstuvwxyzabcdef")
	box2, _ := NewSecretBox(otherKey)

	ciphertext, _ := box1.Encrypt("hunter2")
	if _, err := box2.Decrypt(ciphertext); err == nil {
		t.Fatal("expected decryption with the wrong key to fail")
	}
}
