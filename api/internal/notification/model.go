package notification

import (
	"time"

	"github.com/google/uuid"
)

// kind ของแจ้งเตือน — หน้าเว็บแปลงเป็นข้อความไทยเอง
const (
	KindBookingCreated        = "booking_created"                 // ลูกค้าจองใหม่ → เจ้าของร้าน
	KindBookingUpdated        = "booking_updated"                 // ลูกค้าแก้การจอง → เจ้าของร้าน
	KindBookingCancelled      = "booking_cancelled"               // ลูกค้ายกเลิก → เจ้าของร้าน
	KindCancelledByRestaurant = "booking_cancelled_by_restaurant" // ร้านปิดชั่วคราว/ลบร้าน → ลูกค้า
)

// Notification คือแจ้งเตือนหนึ่งรายการ (ตาราง notifications) — ข้อมูลการจองเป็น snapshot ตอนเกิดเหตุ
type Notification struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"` // ผู้รับ
	Kind           string
	BookingID      uuid.UUID `gorm:"type:uuid"`
	RestaurantID   uuid.UUID `gorm:"type:uuid"`
	RestaurantName string
	CustomerName   string
	BusinessDate   time.Time // วันทำการ (ร้านข้ามคืน ≠ วันปฏิทิน) ใช้ทำลิงก์ไปบอร์ดเจ้าของร้าน
	StartAt        time.Time
	EndAt          time.Time
	PartySize      int
	Reason         string
	ReadAt         *time.Time
	CreatedAt      time.Time
}

// Draft = ส่วนที่ผู้สร้างแจ้งเตือนกำหนดเอง ส่วนที่เหลือ (ชื่อร้าน ชื่อลูกค้า เวลา จำนวนคน) Insert ดึงจาก DB เอง
type Draft struct {
	Recipient    uuid.UUID
	Kind         string
	BookingID    uuid.UUID
	BusinessDate string // YYYY-MM-DD
	Reason       string
}
