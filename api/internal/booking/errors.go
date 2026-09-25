package booking

import (
	"fmt"
	"time"
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

// NotEnoughSeatsError → 409 NOT_ENOUGH_SEATS; At = จุดที่แน่นที่สุด, Available = ที่ว่าง ณ จุดนั้น
type NotEnoughSeatsError struct {
	At        time.Time
	Available int
}

func (e *NotEnoughSeatsError) Error() string {
	return fmt.Sprintf("NOT_ENOUGH_SEATS: %d seats available at %s", e.Available, e.At.Format(time.RFC3339))
}
