package restaurant

import (
	"math"
	"time"

	"jongyoung/internal/booking"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Restaurant struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID             uuid.UUID `gorm:"type:uuid;not null"`
	Name                string
	Description         string
	Cuisine             string
	Address             string
	MapURL              string `gorm:"column:map_url"` // ว่าง = ค้นแผนที่จาก Address
	Seats               int
	OpenMinute          int
	CloseMinute         int
	ClosedWeekdays      int // บิตวันปิดประจำสัปดาห์ (ดู booking.Hours)
	BreakStartMinute    int // ช่วงพัก (ดู booking.Hours) — เท่ากัน = ไม่มีช่วงพัก
	BreakEndMinute      int
	CancelBeforeMinutes int
	RatingSum           int
	RatingCount         int
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt
	Images              []Image `gorm:"foreignKey:RestaurantID"`
}

type Image struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID `gorm:"type:uuid;not null"`
	URL          string    `gorm:"column:url"`
	SortOrder    int
}

func (Image) TableName() string { return "restaurant_images" }

func (r Restaurant) Hours() booking.Hours {
	return booking.Hours{OpenMinute: r.OpenMinute, CloseMinute: r.CloseMinute, ClosedWeekdays: r.ClosedWeekdays,
		BreakStartMinute: r.BreakStartMinute, BreakEndMinute: r.BreakEndMinute}
}

// AverageRating คืน nil เมื่อยังไม่มีรีวิว (หน้าเว็บแสดง "ยังไม่มีรีวิว" แทน 0)
func (r Restaurant) AverageRating() *float64 {
	if r.RatingCount == 0 {
		return nil
	}
	avg := math.Round(float64(r.RatingSum)/float64(r.RatingCount)*10) / 10
	return &avg
}

// Input คือข้อมูลร้านที่เจ้าของกรอก (ใช้ทั้งสร้างและแก้ไข)
type Input struct {
	Name                string
	Description         string
	Cuisine             string
	Address             string
	MapURL              string
	Seats               int
	OpenMinute          int
	CloseMinute         int
	ClosedWeekdays      int
	BreakStartMinute    int
	BreakEndMinute      int
	CancelBeforeMinutes int
}

func (in Input) Hours() booking.Hours {
	return booking.Hours{OpenMinute: in.OpenMinute, CloseMinute: in.CloseMinute, ClosedWeekdays: in.ClosedWeekdays,
		BreakStartMinute: in.BreakStartMinute, BreakEndMinute: in.BreakEndMinute}
}

// ListQuery คือตัวกรองของหน้าค้นหา; Date != nil แปลว่าผู้ใช้เลือกวัน/เวลา/จำนวนคน → คำนวณปุ่มเวลาด้วย
type ListQuery struct {
	Q       string
	Cuisine string
	Sort    string // "rating" | "reviews" | "" (ใหม่สุด)
	Limit   int
	Offset  int
	Date    *time.Time
	Minute  int
	Party   int
}

// ListItem = ร้าน + ปุ่มเวลา 5 ปุ่ม (มีเฉพาะเมื่อค้นด้วยวัน/เวลา)
type ListItem struct {
	Restaurant Restaurant
	Slots      []booking.CardSlot
}

// NextAvailable คือวันทำการถัดไปที่ยังมีช่วงว่างพอ (ใช้ทำ empty state "ว่างวันถัดไป")
type NextAvailable struct {
	Date  time.Time
	Slots []booking.CardSlot
}
