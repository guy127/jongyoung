package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"jongyoung/internal/httputil"

	"github.com/gin-gonic/gin"
)

// RateLimit จำกัดคำขอที่ "เขียนข้อมูล" (POST/PUT/DELETE) ต่อ IP แบบ fixed window
//
// ทำไมจำกัดเฉพาะการเขียน:
//   - การอ่าน (GET) ส่วนใหญ่มาจาก Next.js ฝั่ง server ซึ่งมี IP เดียว ถ้าจำกัดด้วย IP ผู้ใช้ทุกคนจะใช้โควตาเดียวกัน
//   - ที่ต้องกันคือการยิงจอง/รีวิวถี่ ๆ ซึ่งมาจาก browser ของแต่ละคนโดยตรง
//
// fixed window = นับจำนวนคำขอต่อ IP ในช่วงเวลา window พอขึ้นช่วงใหม่ก็ล้างตัวนับทั้งหมด
// ข้อดี: โค้ดสั้น หน่วยความจำไม่โตไม่รู้จบ; ข้อเสีย: รอยต่อระหว่างสองช่วงยิงได้เกือบ 2 เท่า (ยอมรับได้สำหรับการกันแบบเบา ๆ)
// ถ้ารันหลาย instance ตัวนับจะแยกกันต่อเครื่อง — ต้องย้ายไปเก็บที่ Redis เมื่อถึงตอนนั้น
func RateLimit(limit int, window time.Duration, now func() time.Time) gin.HandlerFunc {
	var (
		mu          sync.Mutex
		windowStart = now()
		counts      = map[string]int{}
	)

	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		mu.Lock()
		t := now()
		if t.Sub(windowStart) >= window {
			windowStart = t
			counts = map[string]int{}
		}
		ip := c.ClientIP()
		counts[ip]++
		over := counts[ip] > limit
		retryAfter := windowStart.Add(window).Sub(t)
		mu.Unlock()

		if over {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			httputil.Abort(c, http.StatusTooManyRequests, "RATE_LIMITED", "ทำรายการถี่เกินไป กรุณารอสักครู่", nil)
			return
		}
		c.Next()
	}
}
