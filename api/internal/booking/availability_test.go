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
