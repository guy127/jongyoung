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
