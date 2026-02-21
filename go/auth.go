package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// ── OIDC Configuration ──────────────────────────────────────────────────────

// AuthConfig holds the OIDC provider configuration.
type AuthConfig struct {
	IssuerURL string // e.g. https://auth.spencerbaumruk.com/realms/master
	ClientID  string // e.g. scrabble
}

// UserClaims represents the claims extracted from a verified access token.
type UserClaims struct {
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"preferred_username"`
}

// ── Auth Verifier ───────────────────────────────────────────────────────────

// AuthVerifier validates OIDC access tokens using JWKS discovery.
type AuthVerifier struct {
	verifier *oidc.IDTokenVerifier
	clientID string
}

// NewAuthVerifier creates a verifier by performing OIDC discovery on the issuer.
func NewAuthVerifier(ctx context.Context, cfg AuthConfig) (*AuthVerifier, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		// Access tokens may not have our client ID in the "aud" claim;
		// Keycloak typically sets aud to "account" for access tokens.
		// We verify the "azp" (authorized party) claim instead.
		SkipClientIDCheck: true,
	})

	return &AuthVerifier{
		verifier: verifier,
		clientID: cfg.ClientID,
	}, nil
}

// VerifyToken validates a raw JWT access token and returns the user claims.
func (av *AuthVerifier) VerifyToken(ctx context.Context, rawToken string) (*UserClaims, error) {
	token, err := av.verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	var claims struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Azp               string `json:"azp"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	// Verify the authorized party matches our client ID
	if claims.Azp != "" && claims.Azp != av.clientID {
		return nil, fmt.Errorf("token not issued for this client (azp=%s)", claims.Azp)
	}

	return &UserClaims{
		Subject:  claims.Sub,
		Email:    claims.Email,
		Username: claims.PreferredUsername,
	}, nil
}

// ── Context Helpers ─────────────────────────────────────────────────────────

// getUserClaimsFromContext returns the full user claims from the request context,
// or nil if the user is not authenticated.
func getUserClaimsFromContext(ctx context.Context) *UserClaims {
	if v, ok := ctx.Value(userClaimsContextKey).(*UserClaims); ok {
		return v
	}
	return nil
}

// ── Middleware ───────────────────────────────────────────────────────────────

// extractAuth reads the Bearer token from the Authorization header, validates
// it, and injects the user's ID and claims into the request context.
// If the token is missing or invalid, the request proceeds without user context.
func extractAuth(av *AuthVerifier, r *http.Request) *http.Request {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return r
	}
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := av.VerifyToken(r.Context(), rawToken)
	if err != nil {
		return r
	}
	ctx := context.WithValue(r.Context(), userIDContextKey, claims.Subject)
	ctx = context.WithValue(ctx, userClaimsContextKey, claims)
	return r.WithContext(ctx)
}

// ── Handler ─────────────────────────────────────────────────────────────────

// handleMe returns the authenticated user's claims, or 401 if not authenticated.
func handleMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, 405, "method not allowed")
			return
		}
		claims := getUserClaimsFromContext(r.Context())
		if claims == nil {
			writeError(w, 401, "not authenticated")
			return
		}
		writeJSON(w, 200, claims)
	}
}
