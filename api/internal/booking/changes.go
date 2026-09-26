package booking

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SeatsBelowBookingsError → 409 SEATS_BELOW_EXISTING_BOOKINGS
type SeatsBelowBookingsError struct {
	At   time.Time // จุดที่คนในร้านมากที่สุด
	Peak int       // จำนวนคน ณ จุดนั้น = ที่นั่งขั้นต่ำที่ตั้งได้
}

func (e *SeatsBelowBookingsError) Error() string {
	return fmt.Sprintf("SEATS_BELOW_EXISTING_BOOKINGS: %d people at %s", e.Peak, e.At.Format(time.RFC3339))
}

// HoursConflictError → 409 HOURS_CONFLICT_EXISTING_BOOKINGS
type HoursConflictError struct {
	BookingIDs []uuid.UUID // booking ที่จะตกนอกเวลาเปิดใหม่
}

func (e *HoursConflictError) Error() string {
	return fmt.Sprintf("HOURS_CONFLICT_EXISTING_BOOKINGS: %d bookings outside new hours", len(e.BookingIDs))
}

// CheckSeatReduction: เจ้าของร้านลดที่นั่งได้ก็ต่อเมื่อไม่ต่ำกว่าคนสูงสุดของ booking ที่ยังไม่จบ
// ใช้ maxConcurrent ตัวเดียวกับการจอง — ช่วงที่ตรวจคือตั้งแต่ booking แรกเริ่มจนตัวสุดท้ายจบ
func CheckSeatReduction(future []Booking, newSeats int) error {
	if len(future) == 0 {
		return nil
	}
	start, end := future[0].StartAt, future[0].EndAt
	for _, b := range future {
		if b.StartAt.Before(start) {
			start = b.StartAt
		}
		if b.EndAt.After(end) {
			end = b.EndAt
		}
	}
	if peak, at := maxConcurrent(future, start, end); peak > newSeats {
		return &SeatsBelowBookingsError{At: at, Peak: peak}
	}
	return nil
}

// CheckHoursChange: ทุก booking ที่ยังไม่จบต้องยังอยู่ในเวลาเปิดใหม่ (ใช้ Hours.Fits ตัวเดียวกับการจอง)
func CheckHoursChange(future []Booking, newHours Hours) error {
	var outside []uuid.UUID
	for _, b := range future {
		if !newHours.Fits(b.StartAt, b.EndAt) {
			outside = append(outside, b.ID)
		}
	}
	if len(outside) > 0 {
		return &HoursConflictError{BookingIDs: outside}
	}
	return nil
}

// CardSlot คือปุ่มเวลาบนการ์ดร้านในหน้าค้นหา
type CardSlot struct {
	StartAt   time.Time
	EndAt     time.Time
	Available int
	Closed    bool // ร้านปิดช่วงนั้น (นอกเวลา/ช่วงพัก/ปิดชั่วคราว) หรือจองไม่ทันแล้ว (เลย lead time)
}

// SlotsAround คืน 5 ช่วงรอบเวลาที่ผู้ใช้ค้น (ก่อน 2 ช่วง, ตรงเวลา, หลัง 2 ช่วง) ของวันทำการ date
// minuteOfDay คือเวลาบนนาฬิกาที่ผู้ใช้เลือก — แปลงผ่าน Hours.At จึงรองรับร้านข้ามคืน
func SlotsAround(h Hours, seats int, bookings []Booking, closures []Closure, date time.Time, minuteOfDay int, now time.Time) []CardSlot {
	center := h.At(date, minuteOfDay)
	earliest := now.Add(LeadTime)
	slots := make([]CardSlot, 0, 5)
	for i := -2; i <= 2; i++ {
		start := center.Add(time.Duration(i) * SlotLength)
		end := start.Add(SlotLength)
		s := CardSlot{StartAt: start, EndAt: end}
		if !h.Fits(start, end) || start.Before(earliest) || closureAt(closures, start, end) != nil {
			s.Closed = true
		} else {
			peak, _ := maxConcurrent(bookings, start, end)
			s.Available = seats - peak
		}
		slots = append(slots, s)
	}
	return slots
}
