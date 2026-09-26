package restaurant

import (
	"errors"
	"fmt"
	"time"

	"jongyoung/internal/booking"

	"github.com/google/uuid"
)

// RestaurantRequest ใช้ทั้ง POST (ต้องมี image_urls) และ PUT (ไม่ใช้ image_urls — จัดการรูปผ่าน /images)
type RestaurantRequest struct {
	Name                string   `json:"name" binding:"required,max=120" example:"ครัวบ้านสวน"`
	Description         string   `json:"description" binding:"max=2000"`
	Cuisine             string   `json:"cuisine" binding:"max=60" example:"อาหารไทย"`
	Address             string   `json:"address" binding:"required,max=300"`
	Seats               int      `json:"seats" binding:"required,gt=0,lte=1000" example:"10"`
	OpenTime            string   `json:"open_time" binding:"required" example:"11:00"`
	CloseTime           string   `json:"close_time" binding:"required" example:"22:00"`
	ClosedWeekdays      []int    `json:"closed_weekdays" binding:"omitempty,dive,min=0,max=6" example:"1"` // 0 = อาทิตย์ … 6 = เสาร์
	CancelBeforeMinutes int      `json:"cancel_before_minutes" binding:"required,gte=30,lte=1440" example:"30"`
	ImageURLs           []string `json:"image_urls" binding:"omitempty,max=10,dive,url"`
}

var (
	errInvalidClock  = errors.New(`เวลาเปิด–ปิดต้องเป็นรูปแบบ "HH:MM" และลงที่ :00 หรือ :30`)
	errClosedAllWeek = errors.New("ร้านต้องเปิดอย่างน้อย 1 วันต่อสัปดาห์")
)

// ToInput แปลงเวลา "HH:MM" เป็นนาทีจากเที่ยงคืน — ต้องลง :00/:30 เพราะช่วงจองเป็นช่วงละ 30 นาที
func (req RestaurantRequest) ToInput() (Input, error) {
	open, err := ParseClock(req.OpenTime)
	if err != nil {
		return Input{}, err
	}
	shut, err := ParseClock(req.CloseTime)
	if err != nil {
		return Input{}, err
	}
	closed := 0
	for _, d := range req.ClosedWeekdays {
		closed |= booking.WeekdayMask(time.Weekday(d))
	}
	if closed == booking.WeekdayMask(time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday) {
		return Input{}, errClosedAllWeek
	}
	return Input{
		Name: req.Name, Description: req.Description, Cuisine: req.Cuisine, Address: req.Address,
		Seats: req.Seats, OpenMinute: open, CloseMinute: shut, ClosedWeekdays: closed,
		CancelBeforeMinutes: req.CancelBeforeMinutes,
	}, nil
}

// ParseClock แปลง "HH:MM" (ลง :00 หรือ :30) เป็นนาทีนับจากเที่ยงคืน
func ParseClock(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil || t.Minute()%30 != 0 {
		return 0, errInvalidClock
	}
	return t.Hour()*60 + t.Minute(), nil
}

func formatClock(minute int) string {
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60)
}

type ImageRequest struct {
	URL string `json:"url" binding:"required,url"`
}

type ImageResponse struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	SortOrder int       `json:"sort_order"`
}

type RatingResponse struct {
	Average *float64 `json:"average"` // null = ยังไม่มีรีวิว
	Count   int      `json:"count"`
}

type RestaurantResponse struct {
	ID                  uuid.UUID       `json:"id"`
	OwnerID             uuid.UUID       `json:"owner_id"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	Cuisine             string          `json:"cuisine"`
	Address             string          `json:"address"`
	Seats               int             `json:"seats"`
	OpenTime            string          `json:"open_time" example:"18:00"`
	CloseTime           string          `json:"close_time" example:"02:00"`
	Overnight           bool            `json:"overnight"` // ปิดหลังเที่ยงคืน → หน้าเว็บต้องแสดงป้าย "(เช้าวันที่ n)"
	Open24h             bool            `json:"open_24h"`
	ClosedWeekdays      []int           `json:"closed_weekdays"` // วันปิดประจำสัปดาห์ของวันทำการ (0 = อาทิตย์ … 6 = เสาร์)
	CancelBeforeMinutes int             `json:"cancel_before_minutes"`
	Rating              RatingResponse  `json:"rating"`
	Images              []ImageResponse `json:"images"`
	CreatedAt           time.Time       `json:"created_at"`
}

// SlotResponse ใช้ timestamp เต็มพร้อม offset เสมอ — ห้ามส่ง "00:30" ลอย ๆ (ร้านข้ามคืนเวลาเดียวกันอยู่คนละวัน)
type SlotResponse struct {
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	Available int       `json:"available"`
	Closed    bool      `json:"closed,omitempty"`
}

type ListItemResponse struct {
	RestaurantResponse
	Slots []SlotResponse `json:"slots,omitempty"`
}

type ListResponse struct {
	Items []ListItemResponse `json:"items"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Total int64              `json:"total"`
}

type AvailabilityResponse struct {
	BusinessDate string         `json:"business_date" example:"2026-10-10"`
	Closed       bool           `json:"closed"` // วันปิดประจำสัปดาห์ → slots ว่าง
	OpensAt      time.Time      `json:"opens_at"`
	ClosesAt     time.Time      `json:"closes_at"`
	Seats        int            `json:"seats"`
	Slots        []SlotResponse `json:"slots"`
}

type NextAvailableResponse struct {
	BusinessDate string         `json:"business_date"`
	Slots        []SlotResponse `json:"slots"`
}

func NewRestaurantResponse(r Restaurant) RestaurantResponse {
	closed := []int{} // ส่ง [] ไม่ใช่ null เมื่อเปิดทุกวัน
	for _, d := range r.Hours().ClosedDays() {
		closed = append(closed, int(d))
	}
	images := make([]ImageResponse, len(r.Images))
	for i, img := range r.Images {
		images[i] = ImageResponse{ID: img.ID, URL: img.URL, SortOrder: img.SortOrder}
	}
	return RestaurantResponse{
		ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, Description: r.Description, Cuisine: r.Cuisine,
		Address: r.Address, Seats: r.Seats,
		OpenTime: formatClock(r.OpenMinute), CloseTime: formatClock(r.CloseMinute),
		Overnight: r.CloseMinute < r.OpenMinute, Open24h: r.OpenMinute == r.CloseMinute, ClosedWeekdays: closed,
		CancelBeforeMinutes: r.CancelBeforeMinutes,
		Rating:              RatingResponse{Average: r.AverageRating(), Count: r.RatingCount},
		Images:              images,
		CreatedAt:           r.CreatedAt,
	}
}

func newCardSlots(slots []booking.CardSlot) []SlotResponse {
	out := make([]SlotResponse, len(slots))
	for i, s := range slots {
		out[i] = SlotResponse{StartAt: s.StartAt.In(booking.Bangkok), EndAt: s.EndAt.In(booking.Bangkok), Available: s.Available, Closed: s.Closed}
	}
	return out
}

func newSlots(slots []booking.Slot) []SlotResponse {
	out := make([]SlotResponse, len(slots))
	for i, s := range slots {
		out[i] = SlotResponse{StartAt: s.StartAt.In(booking.Bangkok), EndAt: s.EndAt.In(booking.Bangkok), Available: s.Available}
	}
	return out
}
