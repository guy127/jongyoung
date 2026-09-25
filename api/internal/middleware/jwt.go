// Package middleware มี middleware ตรวจตัวตนของผู้เรียก API
package middleware

import (
	"context"
	"strings"

	"jongyoung/internal/httputil"
	"jongyoung/internal/reqctx"
	"jongyoung/internal/user"

	"github.com/gin-gonic/gin"
)

const bearerPrefix = "Bearer "

// Claims คือข้อมูลที่เราใช้จาก access token ของ Keycloak
type Claims struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

// DisplayName: ใช้ชื่อเต็มก่อน ถ้าไม่มีใช้ username แล้วค่อย email
func (c Claims) DisplayName() string {
	switch {
	case c.Name != "":
		return c.Name
	case c.PreferredUsername != "":
		return c.PreferredUsername
	default:
		return c.Email
	}
}

// TokenVerifier ตรวจ signature + exp + iss + aud แล้วคืน claims (ตัวจริงคือ OIDCVerifier)
type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (Claims, error)
}

// UserProvisioner สร้าง/อัปเดตผู้ใช้ในฐานข้อมูลเราจาก claims (JIT provisioning)
type UserProvisioner interface {
	EnsureExists(ctx context.Context, uid, email, displayName string) (user.User, error)
}

// JWT บังคับให้ต้องมี Bearer token ที่ถูกต้อง แล้วใส่ userID ของฐานข้อมูลเราลง context
// ผู้ใช้ถูกสร้างตรงนี้ตอนเจอ sub ครั้งแรก — login เกิดที่ next-auth ฝั่ง Go จึงไม่เห็น callback
func JWT(verifier TokenVerifier, users UserProvisioner) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			httputil.Unauthorized(c, "ต้องเข้าสู่ระบบ")
			return
		}
		claims, err := verifier.Verify(c.Request.Context(), strings.TrimPrefix(header, bearerPrefix))
		if err != nil || claims.Subject == "" {
			httputil.Unauthorized(c, "token ไม่ถูกต้องหรือหมดอายุ")
			return
		}
		u, err := users.EnsureExists(c.Request.Context(), claims.Subject, claims.Email, claims.DisplayName())
		if err != nil {
			httputil.Internal(c, err)
			return
		}
		c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), u.ID))
		c.Next()
	}
}
