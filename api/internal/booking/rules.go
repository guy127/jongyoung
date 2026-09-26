package booking

import "time"

const (
	SlotLength  = 30 * time.Minute // ช่วงเวลาจองละ 30 นาที
	LeadTime    = 30 * time.Minute // ต้องจองล่วงหน้าอย่างน้อย 30 นาที
	MinDuration = 30 * time.Minute
	MaxDuration = 4 * time.Hour       // กฎเราเอง — กันจองยาวผิดปกติ
	MaxAdvance  = 90 * 24 * time.Hour // จองล่วงหน้าได้ไม่เกิน 90 วัน
)

// Request คือคำขอจองที่แปลงเป็นเวลาจริงแล้ว (ผ่าน Hours.At)
type Request struct {
	StartAt   time.Time
	EndAt     time.Time
	PartySize int
}

// ValidateRequest ตรวจกฎข้อ 5.2 ที่ไม่ต้องใช้ฐานข้อมูล (ข้อ 2–6, 9, 10)
// กฎที่ต้องอ่าน booking อื่น (ข้อ 7, 8) ตรวจใน service ภายใต้ FOR UPDATE
func ValidateRequest(req Request, h Hours, seats int, now time.Time) error {
	if !req.EndAt.After(req.StartAt) {
		return ErrInvalidRange
	}
	if !onHalfHour(req.StartAt) || !onHalfHour(req.EndAt) {
		return ErrNotOnHalfHour
	}
	if d := req.EndAt.Sub(req.StartAt); d < MinDuration || d > MaxDuration {
		return ErrBadDuration
	}
	if !req.StartAt.After(now) {
		return ErrInPast
	}
	if req.StartAt.Before(now.Add(LeadTime)) {
		return ErrTooLateToBook
	}
	if req.StartAt.After(now.Add(MaxAdvance)) {
		return ErrTooFarAhead
	}
	if req.PartySize < 1 {
		return ErrInvalidParty
	}
	if req.PartySize > seats {
		return ErrPartyTooLarge
	}
	// แยก code วันปิดออกจากนอกเวลาเปิด — หน้าเว็บบอกได้ว่า "ร้านปิดทุกวันจันทร์" แทนแค่ "นอกเวลา"
	if date, ok := h.BusinessDate(req.StartAt); ok && h.ClosedOn(date) {
		return ErrClosedWeekday
	}
	if !h.Fits(req.StartAt, req.EndAt) {
		return ErrOutsideHours
	}
	return nil
}

// onHalfHour: เวลาไทยต่างจาก UTC เป็นชั่วโมงเต็ม นาทีจึงเท่ากันทุก timezone ที่เราใช้
func onHalfHour(t time.Time) bool {
	return t.Minute()%30 == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

// CancelDeadline คือเวลาสุดท้ายที่ยังยกเลิก/แก้ไขได้
func CancelDeadline(startAt time.Time, cancelBeforeMinutes int) time.Time {
	return startAt.Add(-time.Duration(cancelBeforeMinutes) * time.Minute)
}

// CanCancel: ยกเลิกได้เมื่อ now <= start - cancel_before (ตรงเส้นตายพอดียังได้)
func CanCancel(startAt time.Time, cancelBeforeMinutes int, now time.Time) bool {
	return !now.After(CancelDeadline(startAt, cancelBeforeMinutes))
}
