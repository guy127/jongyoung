package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"jongyoung/internal/reqctx"
	"jongyoung/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestRouter(v TokenVerifier, p UserProvisioner) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/private", JWT(v, p), func(c *gin.Context) {
		id, _ := reqctx.UserID(c.Request.Context())
		c.String(http.StatusOK, id.String())
	})
	return r
}

func call(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestJWT(t *testing.T) {
	t.Run("ไม่มี token → 401", func(t *testing.T) {
		w := call(newTestRouter(NewMockTokenVerifier(t), NewMockUserProvisioner(t)), "")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), `"code":"UNAUTHORIZED"`)
	})

	t.Run("ไม่ใช่ Bearer → 401", func(t *testing.T) {
		w := call(newTestRouter(NewMockTokenVerifier(t), NewMockUserProvisioner(t)), "Basic abc")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("token ไม่ผ่าน verify → 401 และไม่สร้างผู้ใช้", func(t *testing.T) {
		v := NewMockTokenVerifier(t)
		v.EXPECT().Verify(mock.Anything, "bad").Return(Claims{}, errors.New("expired"))
		w := call(newTestRouter(v, NewMockUserProvisioner(t)), "Bearer bad")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("token ถูก → สร้าง/อัปเดตผู้ใช้จาก claims แล้วใส่ userID ของเราลง context", func(t *testing.T) {
		id := uuid.New()
		v := NewMockTokenVerifier(t)
		v.EXPECT().Verify(mock.Anything, "good").Return(Claims{Subject: "sub-1", Email: "a@b.c", Name: "มะลิ วงศ์ดี"}, nil)
		p := NewMockUserProvisioner(t)
		p.EXPECT().EnsureExists(mock.Anything, "sub-1", "a@b.c", "มะลิ วงศ์ดี").Return(user.User{ID: id}, nil)

		w := call(newTestRouter(v, p), "Bearer good")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, id.String(), w.Body.String())
	})

	t.Run("สร้างผู้ใช้ไม่สำเร็จ → 500", func(t *testing.T) {
		v := NewMockTokenVerifier(t)
		v.EXPECT().Verify(mock.Anything, "good").Return(Claims{Subject: "sub-1"}, nil)
		p := NewMockUserProvisioner(t)
		p.EXPECT().EnsureExists(mock.Anything, "sub-1", "", "").Return(user.User{}, errors.New("db down"))
		w := call(newTestRouter(v, p), "Bearer good")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestClaimsDisplayName(t *testing.T) {
	assert.Equal(t, "มะลิ", Claims{Name: "มะลิ", PreferredUsername: "u", Email: "e"}.DisplayName())
	assert.Equal(t, "u", Claims{PreferredUsername: "u", Email: "e"}.DisplayName())
	assert.Equal(t, "e", Claims{Email: "e"}.DisplayName())
}
