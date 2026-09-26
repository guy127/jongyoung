package notification

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// router จริงที่มีทั้ง /:id/read และ /read-all — ยืนยันว่าสอง route นี้อยู่ร่วมกันได้ใน Gin
func newTestRouter(svc Service, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		if userID != uuid.Nil {
			c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), userID)) // แทน middleware.JWT
		}
	}
	h := NewHandler(svc)
	r.GET("/me/notifications", auth, h.ListMine)
	r.PUT("/me/notifications/read-all", auth, h.MarkAllRead)
	r.PUT("/me/notifications/:id/read", auth, h.MarkRead)
	return r
}

func TestHandler(t *testing.T) {
	t.Run("ไม่มีผู้ใช้ใน context → 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		newTestRouter(NewMockService(t), uuid.Nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me/notifications", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("กดอ่านของคนอื่น → 404", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().MarkRead(mock.Anything, mock.Anything, mock.Anything).Return(ErrNotFound)
		w := httptest.NewRecorder()
		newTestRouter(svc, uuid.New()).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/me/notifications/"+uuid.NewString()+"/read", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("read-all ไม่ถูกตีความเป็น :id → 204", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().MarkAllRead(mock.Anything, mock.Anything).Return(nil)
		w := httptest.NewRecorder()
		newTestRouter(svc, uuid.New()).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/me/notifications/read-all", nil))
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
