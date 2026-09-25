package booking

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bk(party int, start, end time.Time) Booking {
	return Booking{PartySize: party, StartAt: start, EndAt: end, Status: StatusActive}
}

func TestMaxConcurrent(t *testing.T) {
	a := bk(7, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 12, 30))
	b := bk(7, bkk(2026, 10, 10, 12, 30), bkk(2026, 10, 10, 13, 0))
	c := bk(3, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0))

	peak, _ := maxConcurrent(nil, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0))
	assert.Equal(t, 0, peak)

	peak, at := maxConcurrent([]Booking{a, b, c}, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 13, 0))
	assert.Equal(t, 10, peak)
	assert.True(t, at.Equal(bkk(2026, 10, 10, 12, 0)), "peak แรกเกิดตอน 12:00")
}

func TestSlots(t *testing.T) {
	longAgo := bkk(2026, 10, 1, 0, 0)

	t.Run("13) ข้ามคืน date=10 → 18:00 วันที่ 10 ถึงช่วงสุดท้าย 01:30 วันที่ 11", func(t *testing.T) {
		s := Slots(overnight, 10, nil, bkk(2026, 10, 10, 0, 0), longAgo)
		require.Len(t, s, 16)
		assert.True(t, s[0].StartAt.Equal(bkk(2026, 10, 10, 18, 0)))
		assert.True(t, s[15].StartAt.Equal(bkk(2026, 10, 11, 1, 30)))
		assert.True(t, s[15].EndAt.Equal(bkk(2026, 10, 11, 2, 0)))
	})

	t.Run("14) ข้ามคืน date=11 → ไม่มีช่วงตี 0–2 ของเช้าวันที่ 11", func(t *testing.T) {
		s := Slots(overnight, 10, nil, bkk(2026, 10, 11, 0, 0), longAgo)
		require.NotEmpty(t, s)
		for _, slot := range s {
			assert.False(t, slot.StartAt.Before(bkk(2026, 10, 11, 18, 0)), "เจอช่วง %s ซึ่งเป็นของรอบวันที่ 10", slot.StartAt)
		}
	})

	t.Run("ที่ว่างต่อช่วงคิดจาก peak ภายในช่วง", func(t *testing.T) {
		b := bk(4, bkk(2026, 10, 11, 0, 30), bkk(2026, 10, 11, 1, 30))
		s := Slots(overnight, 10, []Booking{b}, bkk(2026, 10, 10, 0, 0), longAgo)
		byStart := map[string]int{}
		for _, slot := range s {
			byStart[slot.StartAt.Format("02 15:04")] = slot.Available
		}
		assert.Equal(t, 10, byStart["11 00:00"])
		assert.Equal(t, 6, byStart["11 00:30"])
		assert.Equal(t, 6, byStart["11 01:00"])
		assert.Equal(t, 10, byStart["11 01:30"])
	})

	t.Run("วันนี้ตัดช่วงที่เริ่มก่อน now+30 นาที แต่เก็บช่วงที่ตรงพอดี", func(t *testing.T) {
		s := Slots(overnight, 10, nil, bkk(2026, 10, 10, 0, 0), bkk(2026, 10, 10, 18, 30))
		require.NotEmpty(t, s)
		assert.True(t, s[0].StartAt.Equal(bkk(2026, 10, 10, 19, 0)))
	})

	t.Run("24 ชม. มี 48 ช่วง", func(t *testing.T) {
		assert.Len(t, Slots(allDay, 10, nil, bkk(2026, 10, 10, 0, 0), longAgo), 48)
	})
}

func TestCheckSeats(t *testing.T) {
	const seats = 10
	s1200, s1230, s1300, s1330 := bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 12, 30), bkk(2026, 10, 10, 13, 0), bkk(2026, 10, 10, 13, 30)

	cases := []struct {
		name       string
		existing   []Booking
		party      int
		start, end time.Time
		wantErr    bool
		available  int
	}{
		{"1) มีคนจอง 7 ขอเพิ่ม 5 ทับกัน → ปฏิเสธ",
			[]Booking{bk(7, s1200, s1300)}, 5, s1230, s1330, true, 3},
		{"2) A7 12:00–12:30, B7 12:30–13:00, C3 12:00–13:00 → ผ่าน (naive SUM พลาด)",
			[]Booking{bk(7, s1200, s1230), bk(7, s1230, s1300)}, 3, s1200, s1300, false, 0},
		{"3a) A7+B3 เต็ม, A แก้เป็น 8 (existing ไม่รวม A) → ปฏิเสธ",
			[]Booking{bk(3, s1200, s1300)}, 8, s1200, s1300, true, 7},
		{"3b) A แก้เป็น 5 → ผ่าน",
			[]Booking{bk(3, s1200, s1300)}, 5, s1200, s1300, false, 0},
		{"11) booking ที่ยกเลิกแล้วไม่ถูกนับ",
			[]Booking{{PartySize: 10, StartAt: s1200, EndAt: s1300, Status: StatusCancelled}}, 10, s1200, s1300, false, 0},
		{"จองเต็มพอดี 10/10 → ผ่าน",
			[]Booking{bk(4, s1200, s1300)}, 6, s1200, s1300, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkSeats(c.existing, seats, c.party, c.start, c.end)
			if !c.wantErr {
				assert.NoError(t, err)
				return
			}
			var nes *NotEnoughSeatsError
			require.True(t, errors.As(err, &nes), "ต้องได้ NotEnoughSeatsError แต่ได้ %v", err)
			assert.Equal(t, c.available, nes.Available)
		})
	}
}
