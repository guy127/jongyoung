// Package httputil รวมรูปแบบ error response กลางและ helper ของ HTTP
package httputil

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody คือ error รูปแบบเดียวทั้งระบบ:
//
//	{ "error": { "code": "NOT_ENOUGH_SEATS", "message": "...", "details": {...} } }
//
// หน้าเว็บตัดสินใจจาก code เท่านั้น — message เป็นภาษาไทยไว้ช่วย debug/แสดงสำรอง
type ErrorBody struct {
	Code    string `json:"code" example:"NOT_ENOUGH_SEATS"`
	Message string `json:"message" example:"ที่นั่งไม่พอ"`
	Details any    `json:"details,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// RequestIDKey คือ key ใน gin.Context ที่ middleware.RequestLog เก็บเลขประจำคำขอไว้
const RequestIDKey = "request_id"

// Abort ตอบ error แล้วหยุด chain ของ handler
func Abort(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: ErrorBody{Code: code, Message: message, Details: details}})
}

// Internal log error จริงไว้ฝั่ง server แล้วตอบผู้ใช้แบบไม่เปิดเผยรายละเอียด
func Internal(c *gin.Context, err error) {
	slog.ErrorContext(c.Request.Context(), "internal error", "path", c.FullPath(), "request_id", c.GetString(RequestIDKey), "error", err)
	Abort(c, http.StatusInternalServerError, "INTERNAL", "เกิดข้อผิดพลาดภายในระบบ", nil)
}

func BadRequest(c *gin.Context, message string) {
	Abort(c, http.StatusBadRequest, "INVALID_REQUEST", message, nil)
}

func Unauthorized(c *gin.Context, message string) {
	Abort(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

func Forbidden(c *gin.Context, code, message string) {
	Abort(c, http.StatusForbidden, code, message, nil)
}

func NotFound(c *gin.Context, message string) {
	Abort(c, http.StatusNotFound, "NOT_FOUND", message, nil)
}
