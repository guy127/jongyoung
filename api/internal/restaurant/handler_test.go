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
