// Package auth implements password hashing, JWT access tokens and the
// in-memory refresh / password-reset token stores.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a bcrypt hash against a plaintext password.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateAccess mints a short-lived HS256 JWT whose subject is the user id.
func GenerateAccess(secret, userID string, ttl time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	})
	return token.SignedString([]byte(secret))
}

// ParseAccess validates an access token and returns the subject (user id).
func ParseAccess(secret, token string) (string, error) {
	var claims jwt.RegisteredClaims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	if claims.Subject == "" {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// RefreshStore keeps opaque refresh tokens in memory.
type refreshEntry struct {
	userID  string
	expires time.Time
}

// RefreshStore is safe for concurrent use.
type RefreshStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]refreshEntry
}

// NewRefreshStore creates a store whose tokens live for ttl.
func NewRefreshStore(ttl time.Duration) *RefreshStore {
	return &RefreshStore{ttl: ttl, items: make(map[string]refreshEntry)}
}

// Create issues a new refresh token for userID.
func (s *RefreshStore) Create(userID string) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[token] = refreshEntry{userID: userID, expires: time.Now().Add(s.ttl)}
	return token, nil
}

// Verify returns the user bound to a live token.
func (s *RefreshStore) Verify(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.items[token]
	if !ok || time.Now().After(entry.expires) {
		delete(s.items, token)
		return "", false
	}
	return entry.userID, true
}

// Rotate swaps a valid token for a fresh one (refresh-token rotation).
func (s *RefreshStore) Rotate(old string) (string, string, bool) {
	userID, ok := s.Verify(old)
	if !ok {
		return "", "", false
	}
	s.mu.Lock()
	delete(s.items, old)
	s.mu.Unlock()
	next, err := s.Create(userID)
	if err != nil {
		return "", "", false
	}
	return next, userID, true
}

// RevokeForUser drops every refresh token owned by userID.
func (s *RefreshStore) RevokeForUser(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, entry := range s.items {
		if entry.userID == userID {
			delete(s.items, token)
		}
	}
}

type resetEntry struct {
	email   string
	expires time.Time
}

// ResetStore keeps short-lived password-reset tokens in memory.
// The token itself is e-mailed (in this MVP: written to the server log).
type ResetStore struct {
	mu    sync.Mutex
	items map[string]resetEntry
}

// NewResetStore creates a store for password-reset tokens.
func NewResetStore() *ResetStore {
	return &ResetStore{items: make(map[string]resetEntry)}
}

// ResetTTL is how long a password-reset token stays valid.
const ResetTTL = 30 * time.Minute

// Create issues a reset token for email.
func (s *ResetStore) Create(email string) (string, error) {
	token, err := randomToken(24)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[token] = resetEntry{email: email, expires: time.Now().Add(ResetTTL)}
	return token, nil
}

// Verify returns the email bound to a live token.
func (s *ResetStore) Verify(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.items[token]
	if !ok || time.Now().After(entry.expires) {
		delete(s.items, token)
		return "", false
	}
	return entry.email, true
}

// Invalidate drops a single token (used after a successful reset).
func (s *ResetStore) Invalidate(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, token)
}

// InvalidateEmail drops every token issued for email.
func (s *ResetStore) InvalidateEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, entry := range s.items {
		if entry.email == email {
			delete(s.items, token)
		}
	}
}
