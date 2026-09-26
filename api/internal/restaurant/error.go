package restaurant

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound        = errors.New("restaurant not found")
	ErrNotOwner        = errors.New("not the owner of this restaurant")
	ErrImageRequired   = errors.New("restaurant must have at least one image")
	ErrImageNotFound   = errors.New("image not found")
	ErrInvalidClosure  = errors.New("ช่วงปิดไม่ถูกต้อง: ระบุ date + start_time + end_time หรือ from_date + to_date, ห้ามย้อนหลัง, ยาวไม่เกิน 90 วัน, ต้องมีเหตุผล (ไม่เกิน 200 ตัวอักษร) และปิดบางช่วงต้องอยู่ในเวลาเปิดของวันทำการนั้น") // → 400 INVALID_CLOSURE
	ErrClosureNotFound = errors.New("closure not found")
)

// AffectsBookingsError → 409 CLOSURE_AFFECTS_BOOKINGS: มีการจองที่จะถูกยกเลิก ต้องยืนยันด้วยรายการนี้ก่อน
type AffectsBookingsError struct {
	Bookings []AffectedBooking
}

func (e *AffectsBookingsError) Error() string {
	return fmt.Sprintf("CLOSURE_AFFECTS_BOOKINGS: %d bookings", len(e.Bookings))
}
