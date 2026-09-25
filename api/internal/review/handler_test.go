package review

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jongyoung/internal/reqctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// handler test: ยิง HTTP จริงเข้า Gin router แล้วเช็ค status + code ที่หน้าเว็บจะใช้
func postReview(t *testing.T, svc Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	userID := uuid.New()
	r.POST("/restaurants/:id/reviews", func(c *gin.Context) {
		c.Request = c.Request.WithContext(reqctx.WithUserID(c.Request.Context(), userID)) // แทน middleware.JWT
	}, NewHandler(svc).Create)
	req := httptest.NewRequest(http.MethodPost, "/restaurants/"+uuid.NewString()+"/reviews", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandlerCreate(t *testing.T) {
	t.Run("คะแนนเกิน 5 → 400 INVALID_REVIEW (ไม่เรียก service)", func(t *testing.T) {
		w := postReview(t, NewMockService(t), `{"score":6}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "INVALID_REVIEW")
	})

	t.Run("รีวิวร้านตัวเอง → 403 OWN_RESTAURANT", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything, 5, "").Return(Review{}, ErrOwnRestaurant)
		w := postReview(t, svc, `{"score":5}`)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "OWN_RESTAURANT")
	})

	t.Run("รีวิวซ้ำ → 409 REVIEW_EXISTS", func(t *testing.T) {
		svc := NewMockService(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything, 4, "x").Return(Review{}, ErrReviewExists)
		w := postReview(t, svc, `{"score":4,"body":"x"}`)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "REVIEW_EXISTS")
	})
}
