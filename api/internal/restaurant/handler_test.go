package restaurant

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jongyoung/internal/booking"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func postClosure(t *testing.T, svc Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/restaurants/:id/closures", func(c *gin.Context) {
		c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), uuid.New())) // แทน middleware.JWT
	}, NewHandler(svc).CreateClosure)
	req := httptest.NewRequest(http.MethodPost, "/restaurants/"+uuid.NewString()+"/closures", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandlerCreateClosure(t *testing.T) {
	body := `{"date":"2026-10-10","start_time":"18:00","end_time":"20:00","reason":"ไฟดับ"}`

	t.Run("ไม่มีเหตุผล → 400 INVALID_CLOSURE (ไม่เรียก service)", func(t *testing.T) {
		w := postClosure(t, NewMockService(t), `{"date":"2026-10-10","start_time":"18:00","end_time":"20:00"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "INVALID_CLOSURE")
	})

	t.Run("ไม่ใช่เจ้าของ → 403", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().CreateClosure(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(booking.Closure{}, ErrNotOwner)
		assert.Equal(t, http.StatusForbidden, postClosure(t, svc, body).Code)
	})

	t.Run("มีการจองทับ → 409 CLOSURE_AFFECTS_BOOKINGS พร้อมรายการ", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().CreateClosure(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(booking.Closure{}, &AffectsBookingsError{Bookings: []AffectedBooking{{CustomerName: "มะลิ"}}})
		w := postClosure(t, svc, body)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "CLOSURE_AFFECTS_BOOKINGS")
		assert.Contains(t, w.Body.String(), "มะลิ")
	})
}

func postImage(t *testing.T, svc Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/restaurants/:id/images", func(c *gin.Context) {
		c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), uuid.New())) // แทน middleware.JWT
	}, NewHandler(svc).AddImage)
	req := httptest.NewRequest(http.MethodPost, "/restaurants/"+uuid.NewString()+"/images", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// รูปแสดงผ่าน next/image ซึ่งตั้งให้โหลดได้เฉพาะ https (web/next.config.ts) — URL แบบอื่นจะกลายเป็นรูปเสีย
func TestHandlerAddImageRequiresHTTPS(t *testing.T) {
	for _, url := range []string{"http://example.com/a.jpg", "javascript:alert(1)", "ftp://example.com/a.jpg"} {
		t.Run(url+" → 400 (ไม่เรียก service)", func(t *testing.T) {
			w := postImage(t, NewMockService(t), `{"url":"`+url+`"}`)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}

	t.Run("https → ผ่านไปถึง service", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().AddImage(mock.Anything, mock.Anything, mock.Anything, "https://example.com/a.jpg").Return(Image{URL: "https://example.com/a.jpg"}, nil)
		assert.Equal(t, http.StatusCreated, postImage(t, svc, `{"url":"https://example.com/a.jpg"}`).Code)
	})
}
