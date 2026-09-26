package booking

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BookingRequest: ผู้ใช้ส่งวันทำการ + เวลาบนนาฬิกา — server แปลงเป็นเวลาจริงเอง
// (เลือกวันที่ 10 เวลา 00:30 ที่ร้านข้ามคืน = 00:30 ของเช้าวันที่ 11)
type BookingRequest struct {
	RestaurantID uuid.UUID `json:"restaurant_id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	Date         string    `json:"date" binding:"required" example:"2026-10-10"`
	StartTime    string    `json:"start_time" binding:"required" example:"18:00"`
	EndTime      string    `json:"end_time" binding:"required" example:"19:00"`
	PartySize    int       `json:"party_size" binding:"required" example:"2"`
}

var errBadChoice = errors.New(`date ต้องเป็น YYYY-MM-DD และเวลาเป็น "HH:MM"`)

func (req BookingRequest) ToChoice() (Choice, error) {
	date, err := ParseDate(req.Date)
	if err != nil {
		return Choice{}, errBadChoice
	}
	start, err := parseClock(req.StartTime)
	if err != nil {
		return Choice{}, err
	}
	end, err := parseClock(req.EndTime)
	if err != nil {
		return Choice{}, err
	}
	return Choice{Date: date, StartMinute: start, EndMinute: end, PartySize: req.PartySize}, nil
}

// parseClock รับ "HH:MM" ใด ๆ — การบังคับ :00/:30 เป็นกฎข้อ 10 ที่ ValidateRequest ตรวจ (ได้ code ที่ชัดกว่า)
func parseClock(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, errBadChoice
	}
	return t.Hour()*60 + t.Minute(), nil
}

type RestaurantSummary struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Overnight bool      `json:"overnight"`
}

type BookingResponse struct {
	ID           uuid.UUID         `json:"id"`
	Code         string            `json:"code" example:"JY-7F3K2A"`
	Restaurant   RestaurantSummary `json:"restaurant"`
	CustomerName string            `json:"customer_name,omitempty"`
	PartySize    int               `json:"party_size"`
	StartAt      time.Time         `json:"start_at"`
	EndAt        time.Time         `json:"end_at"`
	BusinessDate string            `json:"business_date"` // วันทำการของรอบที่ booking นี้อยู่ (ใช้ทำป้าย "(เช้าวันที่ n)")
	Status       string            `json:"status" example:"active"`
	CancelledAt  *time.Time        `json:"cancelled_at,omitempty"`
	CancelUntil  time.Time         `json:"cancel_until"`
	CanChange    bool              `json:"can_change"` // แก้/ยกเลิกได้ตอนนี้ไหม (หน้าเว็บใช้ disable ปุ่มพร้อมเหตุผล)
	CreatedAt    time.Time         `json:"created_at"`
}

// Code = เลขที่จองสั้น ๆ ไว้แสดงที่ร้าน (6 ตัวแรกของ UUID)
func Code(id uuid.UUID) string {
	return "JY-" + strings.ToUpper(strings.ReplaceAll(id.String(), "-", "")[:6])
}

// businessDate: ถ้าเวลาเริ่มอยู่ก่อนเวลาเปิดของวันปฏิทินนั้น แปลว่าเป็นส่วนหลังเที่ยงคืนของรอบเมื่อวาน
func businessDate(start time.Time, h Hours) string {
	local := start.In(Bangkok)
	opensAt, _ := h.Window(local)
	if !h.Is24h() && local.Before(opensAt) {
		local = local.AddDate(0, 0, -1)
	}
	return local.Format("2006-01-02")
}

func NewBookingResponse(v View, now time.Time) BookingResponse {
	h := Hours{OpenMinute: v.OpenMinute, CloseMinute: v.CloseMinute}
	until := CancelDeadline(v.StartAt, v.CancelBeforeMinutes)
	resp := BookingResponse{
		ID:   v.ID,
		Code: Code(v.ID),
		Restaurant: RestaurantSummary{ID: v.RestaurantID, Name: v.RestaurantName, Address: v.RestaurantAddress,
			Overnight: v.CloseMinute < v.OpenMinute},
		CustomerName: v.CustomerName,
		PartySize:    v.PartySize,
		StartAt:      v.StartAt.In(Bangkok),
		EndAt:        v.EndAt.In(Bangkok),
		BusinessDate: businessDate(v.StartAt, h),
		Status:       v.Status,
		CancelUntil:  until.In(Bangkok),
		CanChange:    v.Status == StatusActive && v.StartAt.After(now) && CanCancel(v.StartAt, v.CancelBeforeMinutes, now),
		CreatedAt:    v.CreatedAt,
	}
	if v.CancelledAt != nil {
		t := v.CancelledAt.In(Bangkok)
		resp.CancelledAt = &t
	}
	return resp
}

type BoardSlot struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Booked  int       `json:"booked"` // คนในร้านสูงสุดในช่วงนี้
}

type BoardResponse struct {
	BusinessDate string            `json:"business_date"`
	Closed       bool              `json:"closed"` // วันปิดประจำสัปดาห์
	OpensAt      time.Time         `json:"opens_at"`
	ClosesAt     time.Time         `json:"closes_at"`
	Seats        int               `json:"seats"`
	Slots        []BoardSlot       `json:"slots"`
	Bookings     []BookingResponse `json:"bookings"`
}
