package booking

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RuleError คือกฎที่ตรวจได้โดยไม่ต้องใช้ฐานข้อมูล Code ส่งต่อให้หน้าเว็บแปลเป็นข้อความไทย
type RuleError struct {
	Code    string
	Message string
}

func (e *RuleError) Error() string { return e.Code + ": " + e.Message }

var (
	ErrInvalidRange  = &RuleError{"INVALID_TIME_RANGE", "เวลาสิ้นสุดต้องหลังเวลาเริ่ม"}
	ErrNotOnHalfHour = &RuleError{"NOT_ON_HALF_HOUR", "เวลาเริ่มและสิ้นสุดต้องเป็น :00 หรือ :30"}
	ErrBadDuration   = &RuleError{"INVALID_DURATION", "ระยะเวลาจองต้องอยู่ระหว่าง 30 นาทีถึง 4 ชั่วโมง"}
	ErrInPast        = &RuleError{"BOOKING_IN_PAST", "เวลาที่เลือกผ่านไปแล้ว"}
	ErrTooLateToBook = &RuleError{"TOO_LATE_TO_BOOK", "ต้องจองล่วงหน้าอย่างน้อย 30 นาที"}
	ErrTooFarAhead   = &RuleError{"TOO_FAR_AHEAD", "จองล่วงหน้าได้ไม่เกิน 90 วัน"}
	ErrInvalidParty  = &RuleError{"INVALID_PARTY_SIZE", "จำนวนคนต้องอย่างน้อย 1 คน"}
	ErrPartyTooLarge = &RuleError{"PARTY_TOO_LARGE", "จำนวนคนมากกว่าที่นั่งทั้งร้าน"}
	ErrOutsideHours  = &RuleError{"OUTSIDE_OPENING_HOURS", "ช่วงที่เลือกอยู่นอกเวลาเปิด–ปิดของร้าน"}
)

// error ของ service (ต้องใช้ฐานข้อมูล)
var (
	ErrRestaurantNotFound = errors.New("restaurant not found")
	ErrBookingNotFound    = errors.New("booking not found")
	ErrForbidden          = errors.New("not allowed to access this booking")
	ErrNotRestaurantOwner = errors.New("not the owner of this restaurant")
	ErrBookingCancelled   = errors.New("booking already cancelled")       // 409 BOOKING_CANCELLED
	ErrAlreadyStarted     = errors.New("booking already started or past") // 409 BOOKING_ALREADY_STARTED
)

// DuplicateBookingError → 409 DUPLICATE_BOOKING: ผู้ใช้มีการจองร้านนี้ที่ทับช่วงนี้อยู่แล้ว
type DuplicateBookingError struct {
	BookingID uuid.UUID
}

func (e *DuplicateBookingError) Error() string {
	return "DUPLICATE_BOOKING: overlaps booking " + e.BookingID.String()
}

// CancelWindowError → 403 CANCEL_WINDOW_PASSED: เลยเวลาที่ยกเลิก/แก้ไขได้แล้ว
type CancelWindowError struct {
	Until time.Time
}

func (e *CancelWindowError) Error() string {
	return "CANCEL_WINDOW_PASSED: could cancel until " + e.Until.Format(time.RFC3339)
}

// NotEnoughSeatsError → 409 NOT_ENOUGH_SEATS; At = จุดที่แน่นที่สุด, Available = ที่ว่าง ณ จุดนั้น
type NotEnoughSeatsError struct {
	At        time.Time
	Available int
}

func (e *NotEnoughSeatsError) Error() string {
	return fmt.Sprintf("NOT_ENOUGH_SEATS: %d seats available at %s", e.Available, e.At.Format(time.RFC3339))
}
