package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jongyoung/internal/httputil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clock := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }

	r := gin.New()
	r.Use(RateLimit(2, time.Minute, now))
	r.Any("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	call := func(method, ip string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/x", nil)
		req.RemoteAddr = ip + ":1234"
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("เกินโควตาใน window เดียวกัน → 429 พร้อม Retry-After", func(t *testing.T) {
		assert.Equal(t, http.StatusNoContent, call("POST", "10.0.0.1").Code)
		assert.Equal(t, http.StatusNoContent, call("PUT", "10.0.0.1").Code)
		w := call("DELETE", "10.0.0.1")
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "RATE_LIMITED")
		assert.Equal(t, "61", w.Header().Get("Retry-After"))
	})

	t.Run("คนละ IP มีโควตาของตัวเอง", func(t *testing.T) {
		assert.Equal(t, http.StatusNoContent, call("POST", "10.0.0.2").Code)
	})

	t.Run("GET ไม่ถูกนับ", func(t *testing.T) {
		assert.Equal(t, http.StatusNoContent, call("GET", "10.0.0.1").Code)
	})

	t.Run("ขึ้น window ใหม่ → นับใหม่", func(t *testing.T) {
		clock = clock.Add(time.Minute)
		assert.Equal(t, http.StatusNoContent, call("POST", "10.0.0.1").Code)
	})
}

func TestRequestLogSetsID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLog())
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, c.GetString(httputil.RequestIDKey)) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	id := w.Header().Get(RequestIDHeader)
	assert.Len(t, id, 36) // uuid
	assert.Equal(t, id, w.Body.String())

	// ส่งเลขมาเอง → ใช้เลขเดิม
	w = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(RequestIDHeader, "abc-123")
	r.ServeHTTP(w, req)
	assert.Equal(t, "abc-123", w.Header().Get(RequestIDHeader))
}
