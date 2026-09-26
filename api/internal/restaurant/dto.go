package restaurant

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"jongyoung/internal/booking"

	"github.com/google/uuid"
)

// RestaurantRequest ใช้ทั้ง POST (ต้องมี image_urls) และ PUT (ไม่ใช้ image_urls — จัดการรูปผ่าน /images)
type RestaurantRequest struct {
	Name                string   `json:"name" binding:"required,max=120" example:"ครัวบ้านสวน"`
	Description         string   `json:"description" binding:"max=2000"`
	Cuisine             string   `json:"cuisine" binding:"max=60" example:"อาหารไทย"`
	Address             string   `json:"address" binding:"required,max=300"`
	MapURL              string   `json:"map_url" binding:"max=500" example:"https://maps.app.goo.gl/AbCdEf123"` // ไม่บังคับ
	Seats               int      `json:"seats" binding:"required,gt=0,lte=1000" example:"10"`
	OpenTime            string   `json:"open_time" binding:"required" example:"11:00"`
	CloseTime           string   `json:"close_time" binding:"required" example:"22:00"`
	BreakStart          string   `json:"break_start" example:"14:00"` // ไม่บังคับ — ส่งคู่กับ break_end หรือไม่ส่งเลย
	BreakEnd            string   `json:"break_end" example:"17:00"`
	ClosedWeekdays      []int    `json:"closed_weekdays" binding:"omitempty,dive,min=0,max=6" example:"1"` // 0 = อาทิตย์ … 6 = เสาร์
	CancelBeforeMinutes int      `json:"cancel_before_minutes" binding:"required,gte=30,lte=1440" example:"30"`
	ImageURLs           []string `json:"image_urls" binding:"omitempty,max=10,dive,url"`
}

var (
	errInvalidClock  = errors.New(`เวลาเปิด–ปิดต้องเป็นรูปแบบ "HH:MM" และลงที่ :00 หรือ :30`)
	errClosedAllWeek = errors.New("ร้านต้องเปิดอย่างน้อย 1 วันต่อสัปดาห์")
	errInvalidMapURL = errors.New("ลิงก์แผนที่ต้องเป็นลิงก์ Google Maps (https)")
	errInvalidBreak  = errors.New("ช่วงพักต้องกรอกทั้งเวลาเริ่มและเวลาจบ (ลง :00 หรือ :30) และอยู่ภายในเวลาเปิด–ปิด ไม่ติดขอบ")
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
	breakStart, breakEnd, err := parseBreak(req.BreakStart, req.BreakEnd)
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
	mapURL := strings.TrimSpace(req.MapURL)
	if mapURL != "" && !isGoogleMapsURL(mapURL) {
		return Input{}, errInvalidMapURL
	}
	in := Input{
		Name: req.Name, Description: req.Description, Cuisine: req.Cuisine, Address: req.Address, MapURL: mapURL,
		Seats: req.Seats, OpenMinute: open, CloseMinute: shut, ClosedWeekdays: closed,
		BreakStartMinute: breakStart, BreakEndMinute: breakEnd,
		CancelBeforeMinutes: req.CancelBeforeMinutes,
	}
	if !in.Hours().ValidBreak() {
		return Input{}, errInvalidBreak
	}
	return in, nil
}

// parseBreak: ไม่ส่งทั้งคู่ = ไม่มีช่วงพัก (0, 0); ส่งต้องครบคู่ ลง :00/:30 และไม่เท่ากัน
// (ส่วน "อยู่ข้างในรอบ" ตรวจด้วย Hours.ValidBreak ตัวเดียวกับที่ booking ใช้)
func parseBreak(start, end string) (int, int, error) {
	if start == "" && end == "" {
		return 0, 0, nil
	}
	s, errStart := ParseClock(start)
	e, errEnd := ParseClock(end)
	if errStart != nil || errEnd != nil || s == e {
		return 0, 0, errInvalidBreak
	}
	return s, e, nil
}

// isGoogleMapsURL รับเฉพาะลิงก์ Google Maps แบบ https — ลิงก์นี้ไปอยู่ใน <a href> ที่ลูกค้ากด
// จึงห้ามรับ URL อะไรก็ได้ (javascript:, เว็บหลอก)
// ลิงก์แชร์จากแอปเป็น maps.app.goo.gl; จากเว็บเป็น google.com/maps หรือ google.co.th/maps
func isGoogleMapsURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" {
		return false
	}
	switch u.Hostname() {
	case "maps.app.goo.gl", "maps.google.com", "maps.google.co.th":
		return true
	case "google.com", "www.google.com", "google.co.th", "www.google.co.th":
		return u.Path == "/maps" || strings.HasPrefix(u.Path, "/maps/")
	case "goo.gl":
		return strings.HasPrefix(u.Path, "/maps/")
	}
	return false
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
	MapURL              string          `json:"map_url"` // ว่าง = หน้าเว็บค้นแผนที่จาก address
	Seats               int             `json:"seats"`
	OpenTime            string          `json:"open_time" example:"18:00"`
	CloseTime           string          `json:"close_time" example:"02:00"`
	BreakStart          string          `json:"break_start" example:"15:00"` // "" = ไม่มีช่วงพัก
	BreakEnd            string          `json:"break_end" example:"17:00"`
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
	breakStart, breakEnd := "", ""
	if r.Hours().HasBreak() {
		breakStart, breakEnd = formatClock(r.BreakStartMinute), formatClock(r.BreakEndMinute)
	}
	images := make([]ImageResponse, len(r.Images))
	for i, img := range r.Images {
		images[i] = ImageResponse{ID: img.ID, URL: img.URL, SortOrder: img.SortOrder}
	}
	return RestaurantResponse{
		ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, Description: r.Description, Cuisine: r.Cuisine,
		Address: r.Address, MapURL: r.MapURL, Seats: r.Seats,
		OpenTime: formatClock(r.OpenMinute), CloseTime: formatClock(r.CloseMinute),
		BreakStart: breakStart, BreakEnd: breakEnd,
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

// ClosureRequest: ปิดบางช่วงของวันทำการ (date + start_time + end_time) หรือทั้งวัน/หลายวัน (from_date + to_date) อย่างใดอย่างหนึ่ง
// ส่งครั้งแรกไม่ต้องมี confirm_booking_ids — ถ้ามีการจองทับ API ตอบ 409 พร้อมรายการ แล้วส่งซ้ำพร้อม id ที่เห็น
type ClosureRequest struct {
	Date              string      `json:"date" example:"2026-10-15"`
	StartTime         string      `json:"start_time" example:"18:00"`
	EndTime           string      `json:"end_time" example:"20:00"`
	FromDate          string      `json:"from_date" example:"2026-10-20"`
	ToDate            string      `json:"to_date" example:"2026-10-22"`
	Reason            string      `json:"reason" example:"ไฟดับทั้งซอย"`
	ConfirmBookingIDs []uuid.UUID `json:"confirm_booking_ids"`
}

// ToInput ตรวจรูปแบบ (ช่วงเวลาจริงคำนวณใน service เพราะต้องรู้เวลาเปิดของร้าน)
func (req ClosureRequest) ToInput() (ClosureInput, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > 200 {
		return ClosureInput{}, ErrInvalidClosure
	}
	in := ClosureInput{Reason: reason, ConfirmBookingIDs: req.ConfirmBookingIDs}
	partial := req.Date != "" || req.StartTime != "" || req.EndTime != ""
	days := req.FromDate != "" || req.ToDate != ""
	var err error
	switch {
	case partial && !days:
		in.Partial = true
		if in.Date, err = booking.ParseDate(req.Date); err != nil {
			return ClosureInput{}, ErrInvalidClosure
		}
		if in.StartMinute, err = ParseClock(req.StartTime); err != nil {
			return ClosureInput{}, ErrInvalidClosure
		}
		if in.EndMinute, err = ParseClock(req.EndTime); err != nil || in.EndMinute == in.StartMinute {
			return ClosureInput{}, ErrInvalidClosure
		}
	case days && !partial:
		if in.FromDate, err = booking.ParseDate(req.FromDate); err != nil {
			return ClosureInput{}, ErrInvalidClosure
		}
		if in.ToDate, err = booking.ParseDate(req.ToDate); err != nil || in.ToDate.Before(in.FromDate) {
			return ClosureInput{}, ErrInvalidClosure
		}
	default: // ผสมสองแบบ หรือไม่ส่งช่วงเวลาเลย
		return ClosureInput{}, ErrInvalidClosure
	}
	return in, nil
}

type ClosureResponse struct {
	ID      uuid.UUID `json:"id"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Reason  string    `json:"reason"`
}

func newClosure(c booking.Closure) ClosureResponse {
	return ClosureResponse{ID: c.ID, StartAt: c.StartAt.In(booking.Bangkok), EndAt: c.EndAt.In(booking.Bangkok), Reason: c.Reason}
}

type AffectedBookingResponse struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code" example:"JY-7F3K2A"`
	CustomerName string    `json:"customer_name"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	PartySize    int       `json:"party_size"`
}

func newAffected(list []AffectedBooking) []AffectedBookingResponse {
	out := make([]AffectedBookingResponse, len(list))
	for i, b := range list {
		out[i] = AffectedBookingResponse{ID: b.ID, Code: booking.Code(b.ID), CustomerName: b.CustomerName,
			StartAt: b.StartAt.In(booking.Bangkok), EndAt: b.EndAt.In(booking.Bangkok), PartySize: b.PartySize}
	}
	return out
}
