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
	// ClosedWeekdays = วันปิดประจำสัปดาห์ เก็บเป็น 7 บิตในเลขตัวเดียว: บิตที่ n = ปิดทุก time.Weekday(n)
	// (0 = อาทิตย์ … 6 = เสาร์) — เลือกเลขตัวเดียวแทน slice เพื่อให้ Hours ยังเทียบด้วย != ได้ และเก็บใน DB เป็น int ธรรมดา
	ClosedWeekdays int
}

// WeekdayMask แปลงรายชื่อวันเป็นบิต เช่น จันทร์ + เสาร์ = 0b1000010
func WeekdayMask(days ...time.Weekday) int {
	mask := 0
	for _, d := range days {
		mask |= 1 << d
	}
	return mask
}

// ClosedDays คืนรายชื่อวันปิด เรียงอาทิตย์ → เสาร์ (ทางกลับของ WeekdayMask)
func (h Hours) ClosedDays() []time.Weekday {
	var days []time.Weekday
	for d := time.Sunday; d <= time.Saturday; d++ {
		if h.ClosedWeekdays&(1<<d) != 0 {
			days = append(days, d)
		}
	}
	return days
}

// ClosedOn บอกว่ารอบของวันทำการ date เป็นวันปิดประจำสัปดาห์ไหม
// ผูกกับ "วันทำการ" ไม่ใช่วันปฏิทินของเวลาที่จอง: ร้าน 18:00–02:00 ปิดวันจันทร์
// → ตีหนึ่งเช้าวันจันทร์ยังเปิด (รอบวันอาทิตย์) แต่ตีหนึ่งเช้าวันอังคารปิด (รอบวันจันทร์)
func (h Hours) ClosedOn(date time.Time) bool {
	return h.ClosedWeekdays&(1<<date.In(Bangkok).Weekday()) != 0
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

// OpenAtMinute บอกว่าเวลาบนนาฬิกานี้ร้านเปิดอยู่ไหม (อยู่ในรอบ ไม่ว่าจะก่อนหรือหลังเที่ยงคืน)
// นับนาทีจากเวลาเปิดวนรอบ 24 ชม. แล้วเทียบกับความยาวรอบ — วิธีเดียวกับ Fits
func (h Hours) OpenAtMinute(minuteOfDay int) bool {
	offset := (minuteOfDay - h.OpenMinute + 24*60) % (24 * 60)
	return offset < h.DurationMinutes()
}

// At แปลงเวลาบนนาฬิกา (นาทีจากเที่ยงคืน) ที่ผู้ใช้เลือกในวันทำการ date เป็นเวลาจริง
// ถ้ารอบของร้านข้ามเที่ยงคืน (18:00–02:00, หรือ 24 ชม. ที่เริ่มหลังเที่ยงคืน) เวลาที่น้อยกว่าเวลาเปิด
// = ส่วนหลังเที่ยงคืนของรอบนั้น → บวก 1 วัน
// ร้านที่รอบจบในวันเดียว (11:00–22:00) เวลาก่อนเปิดคือ "ยังไม่เปิด" ของวันเดียวกัน ไม่ต้องบวกวัน
func (h Hours) At(date time.Time, minuteOfDay int) time.Time {
	y, m, d := date.In(Bangkok).Date()
	crossesMidnight := h.OpenMinute+h.DurationMinutes() > 24*60
	if crossesMidnight && minuteOfDay < h.OpenMinute {
		d++
	}
	return time.Date(y, m, d, 0, minuteOfDay, 0, 0, Bangkok)
}

// BusinessDate คืนวันทำการของรอบที่มีเวลา t อยู่ข้างใน (เที่ยงคืนเวลาไทย) — ok=false ถ้า t อยู่นอกเวลาเปิด
// t อาจอยู่ในรอบของวันเดียวกัน หรือส่วนหลังเที่ยงคืนของรอบเมื่อวาน จึงลองสองรอบ
func (h Hours) BusinessDate(t time.Time) (date time.Time, ok bool) {
	y, m, d := t.In(Bangkok).Date()
	for _, back := range []int{0, -1} {
		date = time.Date(y, m, d+back, 0, 0, 0, 0, Bangkok)
		opensAt, closesAt := h.Window(date)
		if !t.Before(opensAt) && t.Before(closesAt) {
			return date, true
		}
	}
	return time.Time{}, false
}

// Fits บอกว่าช่วง [start,end) อยู่ในเวลาเปิดของร้านไหม: start ต้องอยู่ในรอบที่ไม่ใช่วันปิด และจบไม่เกินรอบนั้น
// ยกเว้นร้าน 24 ชม. ที่รอบถัดไปเริ่มทันทีที่รอบนี้จบ → คร่อมได้ถ้ารอบถัดไปไม่ใช่วันปิด
func (h Hours) Fits(start, end time.Time) bool {
	date, ok := h.BusinessDate(start)
	if !ok || h.ClosedOn(date) {
		return false
	}
	_, closesAt := h.Window(date)
	if !end.After(closesAt) {
		return true
	}
	next := date.AddDate(0, 0, 1)
	return h.Is24h() && !h.ClosedOn(next) && !end.After(closesAt.Add(24*time.Hour))
}
