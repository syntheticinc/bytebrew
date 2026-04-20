package http

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"

	pluginpkg "github.com/syntheticinc/bytebrew/engine/pkg/plugin"
)

// HMACVerifier is the default JWT verifier used by the engine.
//
// It validates tokens signed with HS256 using a shared secret. The CE server
// uses this when no plugin.JWTVerifier is supplied. Tokens are expected to
// carry the adminClaims shape produced by AuthHandler.Login — subject, role,
// and tenant_id — though unknown claims are tolerated.
type HMACVerifier struct {
	secret []byte
}

// NewHMACVerifier creates a verifier for HS256-signed JWTs.
func NewHMACVerifier(secret string) *HMACVerifier {
	return &HMACVerifier{secret: []byte(secret)}
}

// Verify parses and validates the token, returning the decoded claims.
//
// Hardening:
//   - `WithValidMethods(["HS256"])` pins the accepted algorithm; the
//     pre-parse hook is kept as a belt-and-braces check.
//   - `WithExpirationRequired()` rejects tokens without an `exp` claim, so a
//     non-expiring token cannot be crafted by a caller with the shared secret.
//   - `ScopeAdmin` is only granted when the `role` claim is literally
//     "admin". Every other token decodes with empty scopes and the enclosing
//     auth middleware decides what to do (CE keeps a "no scopes → admin"
//     fallback; stricter deployments can tighten it).
func (v *HMACVerifier) Verify(token string) (pluginpkg.Claims, error) {
	parsed, err := jwt.Parse(
		token,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return v.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return pluginpkg.Claims{}, fmt.Errorf("parse jwt: %w", err)
	}
	if !parsed.Valid {
		return pluginpkg.Claims{}, fmt.Errorf("invalid jwt")
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return pluginpkg.Claims{}, fmt.Errorf("unexpected claims type")
	}

	out := pluginpkg.Claims{}
	if sub, ok := mapClaims["sub"].(string); ok {
		out.Subject = sub
	}
	if tid, ok := mapClaims["tenant_id"].(string); ok {
		out.TenantID = tid
	}
	// Only the "admin" role grants ScopeAdmin. Non-admin / missing roles
	// leave Scopes == 0; the auth middleware decides how to treat them.
	if role, ok := mapClaims["role"].(string); ok && role == "admin" {
		out.Scopes = ScopeAdmin
	}
	return out, nil
}
