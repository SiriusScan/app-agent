package agent

import (
	"context"
	"crypto/subtle"
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrMissingToken   = errors.New("authentication token is missing")
	ErrInvalidToken   = errors.New("authentication token is invalid")
	ErrTokenRevoked   = errors.New("authentication token has been revoked")
	ErrDuplicateToken = errors.New("token already exists")
	ErrTokenExpired   = errors.New("authentication token has expired")
)

// TokenInfo stores token metadata and expiration
type TokenInfo struct {
	ExpiresAt time.Time
	Metadata  map[string]string
}

// Authenticator handles agent authentication
type Authenticator struct {
	mu     sync.RWMutex
	tokens map[string]TokenInfo // Map of valid tokens to their metadata
}

// NewAuthenticator creates a new Authenticator
func NewAuthenticator() *Authenticator {
	auth := &Authenticator{
		tokens: make(map[string]TokenInfo),
	}

	// Start token cleanup goroutine
	go auth.cleanupExpiredTokens()
	return auth
}

// AddToken adds a new authentication token with optional expiration and metadata
func (a *Authenticator) AddToken(token string, expiration time.Duration, metadata map[string]string) error {
	if token == "" {
		return errors.New("token cannot be empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.tokens[token]; exists {
		return ErrDuplicateToken
	}

	expiresAt := time.Now().Add(expiration)
	if expiration == 0 {
		// No expiration
		expiresAt = time.Time{}
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}

	a.tokens[token] = TokenInfo{
		ExpiresAt: expiresAt,
		Metadata:  metadata,
	}
	return nil
}

// RemoveToken removes an authentication token
func (a *Authenticator) RemoveToken(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.tokens, token)
}

// GetTokenMetadata retrieves metadata associated with a token
func (a *Authenticator) GetTokenMetadata(token string) (map[string]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	info, exists := a.tokens[token]
	if !exists {
		return nil, ErrInvalidToken
	}

	// Return a copy of the metadata
	metadata := make(map[string]string, len(info.Metadata))
	for k, v := range info.Metadata {
		metadata[k] = v
	}
	return metadata, nil
}

// ValidateToken checks if a token is valid and not expired
func (a *Authenticator) ValidateToken(token string) error {
	if token == "" {
		return ErrMissingToken
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	info, valid := a.tokens[token]
	if !valid {
		return ErrInvalidToken
	}

	if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
		return ErrTokenExpired
	}

	return nil
}

// AuthenticateContext extracts and validates the token from the context
func (a *Authenticator) AuthenticateContext(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization token")
	}

	// Compare tokens in constant time to prevent timing attacks
	token := tokens[0]
	a.mu.RLock()
	defer a.mu.RUnlock()

	for validToken, info := range a.tokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) == 1 {
			// Check expiration
			if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
				return status.Error(codes.Unauthenticated, "token has expired")
			}
			return nil
		}
	}

	return status.Error(codes.Unauthenticated, "invalid authorization token")
}

// cleanupExpiredTokens periodically removes expired tokens
func (a *Authenticator) cleanupExpiredTokens() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		a.mu.Lock()
		for token, info := range a.tokens {
			if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
				delete(a.tokens, token)
			}
		}
		a.mu.Unlock()
	}
}
