package booking

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	normal    = Hours{OpenMinute: 11 * 60, CloseMinute: 22 * 60} // 11:00–22:00
	overnight = Hours{OpenMinute: 18 * 60, CloseMinute: 2 * 60}  // 18:00–02:00
	allDay    = Hours{OpenMinute: 0, CloseMinute: 0}             // 24 ชม.
)

func TestBangkokOffsetIsPlus7(t *testing.T) {
	_, offset := bkk(2026, 10, 10, 12, 0).Zone()
	assert.Equal(t, 7*60*60, offset)
}

func TestDurationMinutes(t *testing.T) {
	assert.Equal(t, 11*60, normal.DurationMinutes())
	assert.Equal(t, 8*60, overnight.DurationMinutes())
	assert.Equal(t, 1440, allDay.DurationMinutes(), "open == close ต้องเป็น 24 ชม. ไม่ใช่ 0")
}

func TestParseDate(t *testing.T) {
	d, err := ParseDate("2026-10-10")
	require.NoError(t, err)
	assert.True(t, d.Equal(bkk(2026, 10, 10, 0, 0)))

	_, err = ParseDate("10/10/2026")
	assert.Error(t, err)
}

func TestWindow(t *testing.T) {
	date := bkk(2026, 10, 10, 0, 0)
	cases := []struct {
		name       string
		h          Hours
		open, shut time.Time
	}{
		{"ร้านปกติ วันทำการ = วันปฏิทิน", normal, bkk(2026, 10, 10, 11, 0), bkk(2026, 10, 10, 22, 0)},
		{"ข้ามคืน date=10 → 18:00 วันที่ 10 ถึง 02:00 วันที่ 11", overnight, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 11, 2, 0)},
		{"24 ชม.", allDay, bkk(2026, 10, 10, 0, 0), bkk(2026, 10, 11, 0, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			open, shut := c.h.Window(date)
			assert.True(t, open.Equal(c.open), "opensAt = %s", open)
			assert.True(t, shut.Equal(c.shut), "closesAt = %s", shut)
		})
	}
}

// ข้อ 14: รอบของวันที่ 11 เริ่ม 18:00 วันที่ 11 — ช่วงตี 0–2 เช้าวันที่ 11 เป็นของรอบวันที่ 10 ไปแล้ว
func TestWindowExcludesPreviousNightTail(t *testing.T) {
	opensAt, closesAt := overnight.Window(bkk(2026, 10, 11, 0, 0))
	assert.True(t, opensAt.Equal(bkk(2026, 10, 11, 18, 0)), "opensAt = %s", opensAt)
	assert.True(t, closesAt.Equal(bkk(2026, 10, 12, 2, 0)), "closesAt = %s", closesAt)

	oneAM := bkk(2026, 10, 11, 1, 0)
	assert.True(t, oneAM.Before(opensAt), "ตี 1 เช้าวันที่ 11 ต้องไม่อยู่ในรอบวันที่ 11")
}

func TestBusinessDate(t *testing.T) {
	cases := []struct {
		name   string
		h      Hours
		t      time.Time
		want   time.Time
		wantOK bool
	}{
		{"ร้านปกติ เที่ยงวัน = วันเดียวกัน", normal, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 0, 0), true},
		{"ร้านปกติ ก่อนเปิด = นอกเวลา", normal, bkk(2026, 10, 10, 10, 30), time.Time{}, false},
		{"ข้ามคืน 19:00 วันเสาร์ = รอบวันเสาร์", overnight, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 0, 0), true},
		{"ข้ามคืน ตี 1 เช้าวันอาทิตย์ = รอบวันเสาร์ (บอร์ด owner)", overnight, bkk(2026, 10, 11, 1, 0), bkk(2026, 10, 10, 0, 0), true},
		{"ข้ามคืน 02:00 พอดี = ปิดแล้ว", overnight, bkk(2026, 10, 11, 2, 0), time.Time{}, false},
		{"24 ชม. เที่ยงคืนพอดี = รอบวันใหม่", allDay, bkk(2026, 10, 11, 0, 0), bkk(2026, 10, 11, 0, 0), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := c.h.BusinessDate(c.t)
			assert.Equal(t, c.wantOK, ok)
			if c.wantOK {
				assert.True(t, got.Equal(c.want), "date = %s", got)
			}
		})
	}
}

func TestOpenAtMinute(t *testing.T) {
	assert.False(t, normal.OpenAtMinute(10*60+30), "10:30 ร้าน 11:00–22:00 ยังไม่เปิด")
	assert.True(t, normal.OpenAtMinute(11*60))
	assert.False(t, normal.OpenAtMinute(22*60), "22:00 ปิดแล้ว (ช่วงเปิดคือ [open, close))")
	assert.True(t, overnight.OpenAtMinute(60), "01:00 ร้าน 18:00–02:00 ยังเปิดอยู่")
	assert.False(t, overnight.OpenAtMinute(3*60))
	assert.True(t, allDay.OpenAtMinute(4*60))
}

func TestAt(t *testing.T) {
	date := bkk(2026, 10, 10, 0, 0)
	assert.True(t, overnight.At(date, 30).Equal(bkk(2026, 10, 11, 0, 30)), "00:30 ของวันทำการ 10 = เช้าวันที่ 11")
	assert.True(t, overnight.At(date, 19*60).Equal(bkk(2026, 10, 10, 19, 0)))
	assert.True(t, normal.At(date, 12*60).Equal(bkk(2026, 10, 10, 12, 0)))
	assert.True(t, allDay.At(date, 0).Equal(bkk(2026, 10, 10, 0, 0)))

	// ร้านปกติ: เวลาก่อนเปิด = "ยังไม่เปิด" ของวันเดียวกัน ไม่ใช่เช้าวันถัดไป
	// (บั๊กเดิม: ค้น 10:30 ที่ร้าน 11:00–22:00 แล้วได้ปุ่มเวลาของวันที่ 11)
	assert.True(t, normal.At(date, 10*60+30).Equal(bkk(2026, 10, 10, 10, 30)))
	// ร้านปิดเที่ยงคืนพอดี (10:00–00:00) ไม่ถือว่าข้ามวัน
	closesMidnight := Hours{OpenMinute: 10 * 60, CloseMinute: 0}
	assert.True(t, closesMidnight.At(date, 9*60).Equal(bkk(2026, 10, 10, 9, 0)))
	// ร้าน 24 ชม. ที่รอบเริ่ม 10:00: 09:00 เป็นช่วงท้ายรอบ = เช้าวันถัดไป
	allDayFrom10 := Hours{OpenMinute: 10 * 60, CloseMinute: 10 * 60}
	assert.True(t, allDayFrom10.At(date, 9*60).Equal(bkk(2026, 10, 11, 9, 0)))
}

func TestFits(t *testing.T) {
	closesAt23 := Hours{OpenMinute: 11 * 60, CloseMinute: 23 * 60}
	cases := []struct {
		name       string
		h          Hours
		start, end time.Time
		want       bool
	}{
		{"ปกติ อยู่ในเวลา", normal, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0), true},
		{"ปกติ จบพอดีเวลาปิด", normal, bkk(2026, 10, 10, 21, 0), bkk(2026, 10, 10, 22, 0), true},
		{"ปกติ ก่อนเปิด", normal, bkk(2026, 10, 10, 10, 30), bkk(2026, 10, 10, 11, 30), false},
		{"คาบเกี่ยวเวลาปิด 22:30–23:30 ร้านปิด 23:00", closesAt23, bkk(2026, 10, 10, 22, 30), bkk(2026, 10, 10, 23, 30), false},
		{"ข้ามคืน คร่อมเที่ยงคืน", overnight, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30), true},
		{"ข้ามคืน หลังเที่ยงคืนจบพอดีตีสอง", overnight, bkk(2026, 10, 11, 1, 0), bkk(2026, 10, 11, 2, 0), true},
		{"ข้ามคืน เลยตีสอง", overnight, bkk(2026, 10, 11, 1, 30), bkk(2026, 10, 11, 2, 30), false},
		{"ข้ามคืน ก่อนเปิด", overnight, bkk(2026, 10, 10, 17, 30), bkk(2026, 10, 10, 18, 30), false},
		{"24 ชม. คร่อมเที่ยงคืน", allDay, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.h.Fits(c.start, c.end))
		})
	}
}

// วันปิดประจำสัปดาห์ผูกกับ "วันทำการ" (วันที่รอบเริ่ม) ไม่ใช่วันปฏิทินของเวลาที่จอง
// 2026-10-12 เป็นวันจันทร์
func TestClosedWeekdays(t *testing.T) {
	closedMon := func(h Hours) Hours { h.ClosedWeekdays = WeekdayMask(time.Monday); return h }

	t.Run("WeekdayMask ↔ ClosedDays", func(t *testing.T) {
		assert.Equal(t, 0b1000010, WeekdayMask(time.Monday, time.Saturday))
		assert.Equal(t, []time.Weekday{time.Monday, time.Saturday}, Hours{ClosedWeekdays: 0b1000010}.ClosedDays())
		assert.Empty(t, normal.ClosedDays())
	})
	t.Run("ClosedOn ดูวันในสัปดาห์ของวันทำการ", func(t *testing.T) {
		h := closedMon(normal)
		assert.True(t, h.ClosedOn(bkk(2026, 10, 12, 0, 0)))
		assert.False(t, h.ClosedOn(bkk(2026, 10, 11, 0, 0)))
	})

	cases := []struct {
		name       string
		h          Hours
		start, end time.Time
		want       bool
	}{
		{"ร้านปกติ วันจันทร์ → ปิด", closedMon(normal), bkk(2026, 10, 12, 12, 0), bkk(2026, 10, 12, 13, 0), false},
		{"ร้านปกติ วันอังคาร → เปิด", closedMon(normal), bkk(2026, 10, 13, 12, 0), bkk(2026, 10, 13, 13, 0), true},
		{"ข้ามคืน ตีหนึ่งเช้าวันจันทร์ = รอบวันอาทิตย์ → เปิด", closedMon(overnight), bkk(2026, 10, 12, 1, 0), bkk(2026, 10, 12, 2, 0), true},
		{"ข้ามคืน รอบคืนวันจันทร์ → ปิด", closedMon(overnight), bkk(2026, 10, 12, 19, 0), bkk(2026, 10, 12, 20, 0), false},
		{"ข้ามคืน ตีหนึ่งเช้าวันอังคาร = รอบวันจันทร์ → ปิด", closedMon(overnight), bkk(2026, 10, 13, 1, 0), bkk(2026, 10, 13, 2, 0), false},
		{"24 ชม. วันจันทร์ → ปิด", closedMon(allDay), bkk(2026, 10, 12, 12, 0), bkk(2026, 10, 12, 13, 0), false},
		{"24 ชม. คร่อมเข้าวันจันทร์ → ปิด (ครึ่งหลังตกวันปิด)", closedMon(allDay), bkk(2026, 10, 11, 23, 30), bkk(2026, 10, 12, 0, 30), false},
		{"24 ชม. จบเที่ยงคืนพอดีก่อนวันจันทร์ → เปิด", closedMon(allDay), bkk(2026, 10, 11, 23, 0), bkk(2026, 10, 12, 0, 0), true},
		{"24 ชม. คร่อมออกจากวันจันทร์ → ปิด", closedMon(allDay), bkk(2026, 10, 12, 23, 30), bkk(2026, 10, 13, 0, 30), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.h.Fits(c.start, c.end))
		})
	}
}
