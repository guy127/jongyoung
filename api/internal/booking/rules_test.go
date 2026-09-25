package booking

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateRequest(t *testing.T) {
	now := bkk(2026, 10, 10, 17, 30)
	req := func(party int, start, end time.Time) Request { return Request{StartAt: start, EndAt: end, PartySize: party} }
	cases := []struct {
		name string
		h    Hours
		r    Request
		now  time.Time
		want error
	}{
		{"ปกติผ่าน", normal, req(2, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, nil},
		{"end <= start", normal, req(2, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 19, 0)), now, ErrInvalidRange},
		{"ไม่ลง :00/:30", normal, req(2, bkk(2026, 10, 10, 19, 15), bkk(2026, 10, 10, 20, 15)), now, ErrNotOnHalfHour},
		{"ยาวเกิน 4 ชม.", normal, req(2, bkk(2026, 10, 10, 12, 0), bkk(2026, 10, 10, 16, 30)), bkk(2026, 10, 10, 9, 0), ErrBadDuration},
		{"4) เวลาที่ผ่านไปแล้ว", normal, req(2, bkk(2026, 10, 10, 17, 0), bkk(2026, 10, 10, 18, 0)), now, ErrInPast},
		{"5) lead time อีก 20 นาที → ปฏิเสธ", overnight, req(2, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 19, 0)), bkk(2026, 10, 10, 17, 40), ErrTooLateToBook},
		{"5) lead time อีก 30 นาทีพอดี → ผ่าน", overnight, req(2, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 19, 0)), now, nil},
		{"เกิน 90 วัน", normal, req(2, bkk(2027, 1, 20, 12, 0), bkk(2027, 1, 20, 13, 0)), now, ErrTooFarAhead},
		{"จำนวนคน 0", normal, req(0, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, ErrInvalidParty},
		{"10) party > seats", normal, req(11, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, ErrPartyTooLarge},
		{"party = seats พอดี → ผ่าน", normal, req(10, bkk(2026, 10, 10, 19, 0), bkk(2026, 10, 10, 20, 0)), now, nil},
		{"6) ข้ามคืน คร่อมเที่ยงคืน → ผ่าน", overnight, req(2, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30)), now, nil},
		{"6) ข้ามคืน เลยตีสอง → ปฏิเสธ", overnight, req(2, bkk(2026, 10, 11, 1, 30), bkk(2026, 10, 11, 2, 30)), now, ErrOutsideHours},
		{"7) 24 ชม. คร่อมเที่ยงคืน → ผ่าน", allDay, req(2, bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30)), now, nil},
		{"8) คาบเกี่ยวเวลาปิด → ปฏิเสธ", normal, req(2, bkk(2026, 10, 10, 21, 30), bkk(2026, 10, 10, 22, 30)), now, ErrOutsideHours},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateRequest(c.r, c.h, 10, c.now)
			if c.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, c.want)
			}
		})
	}
}

func TestCanCancel(t *testing.T) {
	start := bkk(2026, 10, 10, 18, 0)
	assert.True(t, CancelDeadline(start, 30).Equal(bkk(2026, 10, 10, 17, 30)))
	assert.True(t, CanCancel(start, 30, bkk(2026, 10, 10, 17, 0)), "ก่อนเส้นตาย")
	assert.True(t, CanCancel(start, 30, bkk(2026, 10, 10, 17, 30)), "9) ตรงเส้นตายพอดี → ยังยกเลิกได้")
	assert.False(t, CanCancel(start, 30, bkk(2026, 10, 10, 17, 31)), "9) ช้ากว่าเส้นตาย → ปฏิเสธ")
	assert.False(t, CanCancel(start, 60, bkk(2026, 10, 10, 17, 30)), "ร้านที่ให้ยกเลิกก่อน 60 นาที")
}
