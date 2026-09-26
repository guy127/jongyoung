package middleware

import (
	"log/slog"
	"time"

	"jongyoung/internal/httputil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader ส่งกลับทุก response — ผู้ใช้แจ้งปัญหาพร้อมเลขนี้ แล้วค้น log ด้วยเลขเดียวกันได้ทันที
const RequestIDHeader = "X-Request-ID"

// RequestLog ให้เลขประจำคำขอ แล้ว log หนึ่งบรรทัดต่อคำขอเมื่อจบ (แทน gin.Logger ที่เป็นข้อความธรรมดา)
// log เป็น JSON ผ่าน slog: method, path, status, latency, request_id — ไม่มี header/body จึงไม่หลุด token
func RequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ใช้เลขที่ส่งมาถ้ามี (เช่น proxy ข้างหน้าสร้างไว้แล้ว) ไม่งั้นสร้างใหม่
		id := c.GetHeader(RequestIDHeader)
		if id == "" || len(id) > 64 {
			id = uuid.NewString()
		}
		c.Set(httputil.RequestIDKey, id)
		c.Header(RequestIDHeader, id)

		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		slog.Log(c.Request.Context(), level, "request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path, // ไม่ log query string — อาจมีข้อมูลส่วนตัว
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"request_id", id,
		)
	}
}
