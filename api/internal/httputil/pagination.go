package httputil

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultLimit = 12
	maxLimit     = 50
)

// Page คือ ?page=&limit= ที่ผ่านการตรวจแล้ว (ค่าผิดจะถูกปรับเป็นค่าเริ่มต้น ไม่ตอบ 400)
type Page struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func (p Page) Offset() int { return (p.Page - 1) * p.Limit }

func ParsePage(c *gin.Context) Page {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return Page{Page: page, Limit: limit}
}

// PageMeta ส่งคู่กับรายการเพื่อให้หน้าเว็บทำ pagination ได้
type PageMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}
