package booking

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusCancelled = "cancelled"
)

// Booking คือการจองหนึ่งรายการ (ตาราง bookings)
type Booking struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID `gorm:"type:uuid;not null"`
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	PartySize    int       `gorm:"not null"`
	StartAt      time.Time `gorm:"not null"`
	EndAt        time.Time `gorm:"not null"`
	Status       string    `gorm:"not null;default:active"`
	CancelledAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Closure คือช่วงที่ร้านปิดชั่วคราว (ตาราง restaurant_closures)
// อยู่ใน package booking เพราะการจองและตารางเวลาว่างต้องใช้ — package restaurant เป็นคนสร้าง/ลบ
type Closure struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID `gorm:"type:uuid;not null"`
	StartAt      time.Time
	EndAt        time.Time
	Reason       string
	CreatedAt    time.Time
}

func (Closure) TableName() string { return "restaurant_closures" }
