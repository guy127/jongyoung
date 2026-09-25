package booking

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSeatReduction(t *testing.T) {
	future := []Booking{
		bk(7, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)),
		bk(3, bkk(2026, 10, 10, 19, 30), bkk(2026, 10, 10, 20, 30)),
	}

	t.Run("ลดต่ำกว่า peak → SeatsBelowBookingsError พร้อมเวลาและยอดสูงสุด", func(t *testing.T) {
		err := CheckSeatReduction(future, 9)
		var e *SeatsBelowBookingsError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, 10, e.Peak)
		assert.True(t, e.At.Equal(bkk(2026, 10, 10, 19, 30)))
	})
	t.Run("เท่ากับ peak พอดี → ผ่าน", func(t *testing.T) {
		assert.NoError(t, CheckSeatReduction(future, 10))
	})
	t.Run("ไม่มี booking → ลดได้", func(t *testing.T) {
		assert.NoError(t, CheckSeatReduction(nil, 1))
	})
}

func TestCheckHoursChange(t *testing.T) {
	late := bk(2, bkk(2026, 10, 10, 21, 0), bkk(2026, 10, 10, 22, 0))
	late.ID = uuid.New()
	early := bk(2, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0))
	early.ID = uuid.New()

	t.Run("ย่นปิดเป็น 21:00 → booking 21:00–22:00 ตกนอกเวลา", func(t *testing.T) {
		err := CheckHoursChange([]Booking{early, late}, Hours{OpenMinute: 11 * 60, CloseMinute: 21 * 60})
		var e *HoursConflictError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, []uuid.UUID{late.ID}, e.BookingIDs)
	})
	t.Run("ขยายเวลา → ผ่าน", func(t *testing.T) {
		assert.NoError(t, CheckHoursChange([]Booking{early, late}, Hours{OpenMinute: 10 * 60, CloseMinute: 23 * 60}))
	})
	t.Run("เปลี่ยนเป็น 24 ชม. → ผ่าน", func(t *testing.T) {
		assert.NoError(t, CheckHoursChange([]Booking{early, late}, Hours{}))
	})
}

func TestSlotsAround(t *testing.T) {
	longAgo := bkk(2026, 10, 1, 0, 0)
	date := bkk(2026, 10, 10, 0, 0)

	t.Run("5 ช่วงรอบเวลาที่ค้น ช่วงนอกเวลาเปิดเป็น closed", func(t *testing.T) {
		// ร้านเปิด 18:00 ค้น 18:30 → 17:30, 18:00, 18:30, 19:00, 19:30
		s := SlotsAround(overnight, 10, nil, date, 18*60+30, longAgo)
		require.Len(t, s, 5)
		assert.True(t, s[0].StartAt.Equal(bkk(2026, 10, 10, 17, 30)))
		assert.True(t, s[0].Closed, "17:30 ร้านยังไม่เปิด")
		assert.False(t, s[1].Closed)
		assert.Equal(t, 10, s[1].Available)
	})

	t.Run("ค้นเวลาหลังเที่ยงคืนของรอบข้ามคืน → ได้ช่วงของเช้าวันถัดไป", func(t *testing.T) {
		s := SlotsAround(overnight, 10, nil, date, 60, longAgo) // 01:00 ของรอบวันที่ 10
		assert.True(t, s[2].StartAt.Equal(bkk(2026, 10, 11, 1, 0)))
		assert.False(t, s[2].Closed)
		assert.True(t, s[4].Closed, "02:00 ร้านปิดแล้ว")
	})

	t.Run("ช่วงที่เลย lead time แล้วเป็น closed", func(t *testing.T) {
		s := SlotsAround(normal, 10, nil, date, 12*60, bkk(2026, 10, 10, 11, 45))
		assert.True(t, s[0].Closed, "11:00 ผ่านไปแล้ว")
		assert.True(t, s[1].Closed, "11:30 ผ่านไปแล้ว")
		assert.True(t, s[2].Closed, "12:00 เหลือแค่ 15 นาที น้อยกว่า lead time")
		assert.False(t, s[3].Closed, "12:30 ยังจองได้")
	})

	t.Run("ที่ว่างนับจาก booking", func(t *testing.T) {
		b := bk(8, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0))
		s := SlotsAround(overnight, 10, []Booking{b}, date, 19*60, longAgo)
		assert.Equal(t, 2, s[2].Available)
		assert.Equal(t, 10, s[4].Available)
	})
}
