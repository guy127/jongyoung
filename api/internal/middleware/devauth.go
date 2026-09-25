package middleware

import (
	"log/slog"

	"jongyoung/internal/httputil"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
)

// DevAuth ข้ามการตรวจ token — ใช้ตอน dev เท่านั้น (เปิดด้วย DEV_AUTH=true และต้องเป็น APP_ENV=development)
// ผู้เรียกระบุตัวตนด้วย header X-Dev-User: <keycloak sub> เช่น sub ของ owner1 ใน realm
func DevAuth(users UserProvisioner) gin.HandlerFunc {
	slog.Warn("⚠️ DevAuth เปิดอยู่: API ไม่ตรวจ token — ห้ามใช้บน production")
	return func(c *gin.Context) {
		uid := c.GetHeader("X-Dev-User")
		if uid == "" {
			httputil.Unauthorized(c, "DevAuth: ต้องส่ง header X-Dev-User")
			return
		}
		u, err := users.EnsureExists(c.Request.Context(), uid, uid+"@dev.local", uid)
		if err != nil {
			httputil.Internal(c, err)
			return
		}
		c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), u.ID))
		c.Next()
	}
}
