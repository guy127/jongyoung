package middleware

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCVerifier ใช้ go-oidc ตรวจ access token ของ Keycloak
// ใช้ RemoteKeySet (ดึง JWKS ตอนตรวจครั้งแรกแล้ว cache/rotate เอง) แทน discovery ตอนเริ่ม
// api จึงไม่ต้องรอ Keycloak บูตเสร็จก่อนถึงจะเริ่มได้
type OIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
}

// NewOIDCVerifier: ตั้ง ClientID = audience ทำให้ go-oidc ตรวจ aud ให้ด้วย (พร้อม signature, exp, iss)
func NewOIDCVerifier(ctx context.Context, issuer, audience string) *OIDCVerifier {
	keySet := oidc.NewRemoteKeySet(ctx, issuer+"/protocol/openid-connect/certs")
	return &OIDCVerifier{
		verifier: oidc.NewVerifier(issuer, keySet, &oidc.Config{ClientID: audience}),
	}
}

func (o *OIDCVerifier) Verify(ctx context.Context, rawToken string) (Claims, error) {
	token, err := o.verifier.Verify(ctx, rawToken)
	if err != nil {
		return Claims{}, fmt.Errorf("verify token: %w", err)
	}
	var claims Claims
	if err := token.Claims(&claims); err != nil {
		return Claims{}, fmt.Errorf("parse claims: %w", err)
	}
	return claims, nil
}
