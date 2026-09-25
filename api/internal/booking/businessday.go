package booking

import (
	"time"
	_ "time/tzdata" // ฝัง tzdata ในไบนารี — container distroless หา Asia/Bangkok เจอเสมอ
)

// Bangkok คือ timezone ของทุกร้าน (ไม่มี DST)
var Bangkok = mustLoadLocation("Asia/Bangkok")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

const minutesPerDay = 1440

// Hours คือเวลาเปิด–ปิดของร้าน เป็นนาทีนับจากเที่ยงคืนเวลาไทย
// close <= open แปลว่าเปิดข้ามเที่ยงคืน, open == close แปลว่าเปิด 24 ชม.
type Hours struct {
	OpenMinute  int
	CloseMinute int
}

// DurationMinutes คือความยาวของรอบเปิดหนึ่งรอบ
func (h Hours) DurationMinutes() int {
	d := (h.CloseMinute - h.OpenMinute + minutesPerDay) % minutesPerDay
	if d == 0 {
		d = minutesPerDay // ⚠️ ร้าน 24 ชม. — ถ้าไม่มีบรรทัดนี้จะจองไม่ได้เลย
	}
	return d
}

func (h Hours) Is24h() bool { return h.OpenMinute == h.CloseMinute }

// ParseDate แปลง "YYYY-MM-DD" เป็นเที่ยงคืนของวันนั้นเวลาไทย
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, Bangkok)
}

// Window คืนรอบเปิดของวันทำการ date เป็นช่วง [opensAt, closesAt)
func (h Hours) Window(date time.Time) (opensAt, closesAt time.Time) {
	y, m, d := date.In(Bangkok).Date()
	opensAt = time.Date(y, m, d, 0, h.OpenMinute, 0, 0, Bangkok)
	closesAt = opensAt.Add(time.Duration(h.DurationMinutes()) * time.Minute)
	return opensAt, closesAt
}

// At แปลงเวลาบนนาฬิกา (นาทีจากเที่ยงคืน) ที่ผู้ใช้เลือกในวันทำการ date เป็นเวลาจริง
// เวลาที่น้อยกว่าเวลาเปิด = ส่วนหลังเที่ยงคืนของรอบนั้น → บวก 1 วัน
func (h Hours) At(date time.Time, minuteOfDay int) time.Time {
	y, m, d := date.In(Bangkok).Date()
	if minuteOfDay < h.OpenMinute {
		d++
	}
	return time.Date(y, m, d, 0, minuteOfDay, 0, 0, Bangkok)
}

// Fits บอกว่าช่วง [start,end) อยู่ภายในรอบเปิดรอบเดียวของร้านไหม
// start อาจอยู่ในรอบของวันเดียวกัน หรือส่วนหลังเที่ยงคืนของรอบเมื่อวาน จึงลองสองรอบ
func (h Hours) Fits(start, end time.Time) bool {
	if h.Is24h() {
		return true
	}
	for _, back := range []int{0, -1} {
		opensAt, closesAt := h.Window(start.In(Bangkok).AddDate(0, 0, back))
		if !start.Before(opensAt) && start.Before(closesAt) && !end.After(closesAt) {
			return true
		}
	}
	return false
}
