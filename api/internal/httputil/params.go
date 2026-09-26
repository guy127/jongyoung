package httputil

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PathID อ่าน path parameter ที่เป็น uuid — รูปแบบผิดตอบ 404 เลย (id ที่ไม่ใช่ uuid ไม่มีทางมีอยู่จริง)
// คืน false เมื่อตอบ error ไปแล้ว handler แค่ return ต่อ
func PathID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		NotFound(c, "ไม่พบข้อมูล")
		return uuid.Nil, false
	}
	return id, true
}

// QueryInt อ่าน query parameter ที่เป็นจำนวนเต็มบวก — ไม่มีหรือผิดรูปแบบใช้ค่า fallback
func QueryInt(c *gin.Context, name string, fallback int) int {
	v, err := strconv.Atoi(c.Query(name))
	if err != nil || v < 1 {
		return fallback
	}
	return v
}
