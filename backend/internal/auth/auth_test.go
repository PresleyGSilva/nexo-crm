package auth

import (
	"testing"
	"time"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("supersecret1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "supersecret1" {
		t.Fatal("password stored in plaintext")
	}
	if err := CheckPassword(hash, "supersecret1"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	if err := CheckPassword(hash, "wrongpass1"); err == nil {
		t.Fatal("wrong password accepted")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	token, err := GenerateAccess("test-secret", "user-123", 15*time.Minute)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sub, err := ParseAccess("test-secret", token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sub != "user-123" {
		t.Fatalf("subject = %q, want user-123", sub)
	}
}

func TestAccessTokenRejectsWrongSecret(t *testing.T) {
	token, _ := GenerateAccess("secret-a", "user-1", time.Minute)
	if _, err := ParseAccess("secret-b", token); err == nil {
		t.Fatal("token signed with another secret accepted")
	}
	if _, err := ParseAccess("secret-a", "not-a-token"); err == nil {
		t.Fatal("garbage token accepted")
	}
}

func TestAccessTokenExpiry(t *testing.T) {
	token, _ := GenerateAccess("s", "user-1", -time.Minute)
	if _, err := ParseAccess("s", token); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestRefreshStoreRotate(t *testing.T) {
	store := NewRefreshStore(time.Hour)
	first, err := store.Create("user-9")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if user, ok := store.Verify(first); !ok || user != "user-9" {
		t.Fatalf("verify = %q,%v", user, ok)
	}
	second, user, ok := store.Rotate(first)
	if !ok || user != "user-9" {
		t.Fatalf("rotate = %q,%v", second, ok)
	}
	if _, ok := store.Verify(first); ok {
		t.Fatal("old token still valid after rotation")
	}
	if _, ok := store.Verify(second); !ok {
		t.Fatal("rotated token rejected")
	}
}

func TestResetStoreLifecycle(t *testing.T) {
	store := NewResetStore()
	token, err := store.Create("a@b.c")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if email, ok := store.Verify(token); !ok || email != "a@b.c" {
		t.Fatalf("verify = %q,%v", email, ok)
	}
	store.Invalidate(token)
	if _, ok := store.Verify(token); ok {
		t.Fatal("invalidated token still valid")
	}
	if _, ok := store.Verify("missing"); ok {
		t.Fatal("unknown token accepted")
	}
}
