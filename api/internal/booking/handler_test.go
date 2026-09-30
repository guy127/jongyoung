package booking

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jongyoung/internal/middleware"
	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var handlerNow = time.Date(2026, 10, 10, 9, 0, 0, 0, Bangkok)

// newTestRouter ต่อ route แบบเดียวกับ cmd/api/router.go
// userID = uuid.Nil → ใช้ middleware.JWT ตัวจริง (ไม่มี header จึงตอบ 401 ก่อนถึง verifier)
// userID อื่น → ใส่ผู้ใช้ลง context แทน middleware.JWT
func newTestRouter(svc Service, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := middleware.JWT(nil, nil)
	if userID != uuid.Nil {
		auth = func(c *gin.Context) {
			c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), userID))
		}
	}
	h := NewHandler(svc, func() time.Time { return handlerNow })
	bookings := r.Group("/bookings", auth)
	bookings.POST("", h.Create)
	bookings.DELETE("/:id", h.Cancel)
	return r
}

func postBooking(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// errorBody = รูปแบบ error กลางของ httputil — หน้าเว็บตัดสินใจจาก code และ details
type errorBody struct {
	Error struct {
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var body errorBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestHandler(t *testing.T) {
	validBody := `{"restaurant_id":"` + uuid.NewString() + `","date":"2026-10-10","start_time":"18:00","end_time":"19:00","party_size":2}`

	t.Run("ไม่มี token → 401 (service ไม่ถูกเรียก)", func(t *testing.T) {
		w := postBooking(newTestRouter(NewMockService(t), uuid.Nil), validBody)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("body ไม่ครบ → 400 (service ไม่ถูกเรียก)", func(t *testing.T) {
		w := postBooking(newTestRouter(NewMockService(t), uuid.New()), `{"date":"2026-10-10"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ที่นั่งไม่พอ → 409 NOT_ENOUGH_SEATS พร้อมที่นั่งที่เหลือและเวลาที่ชน", func(t *testing.T) {
		at := time.Date(2026, 10, 10, 18, 30, 0, 0, Bangkok)
		svc := NewMockService(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(Booking{}, &NotEnoughSeatsError{At: at, Available: 3})

		w := postBooking(newTestRouter(svc, uuid.New()), validBody)

		assert.Equal(t, http.StatusConflict, w.Code)
		body := decodeError(t, w)
		assert.Equal(t, "NOT_ENOUGH_SEATS", body.Error.Code)
		assert.EqualValues(t, 3, body.Error.Details["available"])
		assert.Equal(t, "2026-10-10T18:30:00+07:00", body.Error.Details["at"])
	})

	t.Run("จองซ้อนตัวเอง → 409 DUPLICATE_BOOKING พร้อม id ของรายการเดิม (ไม่ใช่ NOT_ENOUGH_SEATS)", func(t *testing.T) {
		existing := uuid.New()
		svc := NewMockService(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(Booking{}, &DuplicateBookingError{BookingID: existing})

		w := postBooking(newTestRouter(svc, uuid.New()), validBody)

		assert.Equal(t, http.StatusConflict, w.Code)
		body := decodeError(t, w)
		assert.Equal(t, "DUPLICATE_BOOKING", body.Error.Code)
		assert.Equal(t, existing.String(), body.Error.Details["booking_id"])
	})

	t.Run("ยกเลิกการจองของคนอื่น → 403 NOT_BOOKING_OWNER", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().Cancel(mock.Anything, mock.Anything, mock.Anything).Return(ErrForbidden)

		w := httptest.NewRecorder()
		newTestRouter(svc, uuid.New()).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/bookings/"+uuid.NewString(), nil))

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Equal(t, "NOT_BOOKING_OWNER", decodeError(t, w).Error.Code)
	})

	t.Run("ยกเลิกช้าเกิน → 403 CANCEL_WINDOW_PASSED พร้อมเวลาที่ยกเลิกได้ถึง", func(t *testing.T) {
		until := time.Date(2026, 10, 10, 17, 30, 0, 0, Bangkok)
		svc := NewMockService(t)
		svc.EXPECT().Cancel(mock.Anything, mock.Anything, mock.Anything).Return(&CancelWindowError{Until: until})

		w := httptest.NewRecorder()
		newTestRouter(svc, uuid.New()).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/bookings/"+uuid.NewString(), nil))

		assert.Equal(t, http.StatusForbidden, w.Code)
		body := decodeError(t, w)
		assert.Equal(t, "CANCEL_WINDOW_PASSED", body.Error.Code)
		assert.Equal(t, "2026-10-10T17:30:00+07:00", body.Error.Details["cancel_until"])
	})
}
