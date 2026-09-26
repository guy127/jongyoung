package restaurant

import (
	"context"
	"errors"
	"testing"
	"time"

	"jongyoung/internal/booking"
	"jongyoung/internal/notification"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2026, 10, 10, 9, 0, 0, 0, booking.Bangkok)

func newServiceWith(repo *MockRepository) *service {
	return NewService(repo, func() time.Time { return fixedNow })
}

// expectTx ให้ Transaction เรียก fn ด้วย mock ตัวเดิม (จำลองว่าอยู่ในทรานแซกชัน)
func expectTx(repo *MockRepository) {
	repo.EXPECT().Transaction(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(Repository) error) error { return fn(repo) })
}

func baseInput() Input {
	return Input{Name: "ครัวบ้านสวน", Address: "กรุงเทพฯ", Seats: 10, OpenMinute: 11 * 60, CloseMinute: 22 * 60, CancelBeforeMinutes: 30}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	owner := uuid.New()

	t.Run("ไม่มีรูป → ErrImageRequired", func(t *testing.T) {
		_, err := newServiceWith(NewMockRepository(t)).Create(ctx, owner, baseInput(), nil)
		assert.ErrorIs(t, err, ErrImageRequired)
	})

	t.Run("สร้างร้านพร้อมรูปตามลำดับ และผู้สร้างเป็นเจ้าของ", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().Create(ctx, mock.MatchedBy(func(r *Restaurant) bool {
			return r.OwnerID == owner && len(r.Images) == 2 && r.Images[1].SortOrder == 1
		})).Return(nil)
		rest, err := newServiceWith(repo).Create(ctx, owner, baseInput(), []string{"https://a/1.jpg", "https://a/2.jpg"})
		require.NoError(t, err)
		assert.Equal(t, owner, rest.OwnerID)
	})
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	owner, id := uuid.New(), uuid.New()
	existing := Restaurant{ID: id, OwnerID: owner, Seats: 10, OpenMinute: 11 * 60, CloseMinute: 22 * 60}
	booked := []booking.Booking{
		{ID: uuid.New(), PartySize: 7, StartAt: time.Date(2026, 10, 10, 19, 0, 0, 0, booking.Bangkok), EndAt: time.Date(2026, 10, 10, 20, 0, 0, 0, booking.Bangkok), Status: booking.StatusActive},
		{ID: uuid.New(), PartySize: 3, StartAt: time.Date(2026, 10, 10, 21, 0, 0, 0, booking.Bangkok), EndAt: time.Date(2026, 10, 10, 22, 0, 0, 0, booking.Bangkok), Status: booking.StatusActive},
	}

	t.Run("ไม่ใช่เจ้าของ → ErrNotOwner", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(existing, nil)
		_, err := newServiceWith(repo).Update(ctx, uuid.New(), id, baseInput())
		assert.ErrorIs(t, err, ErrNotOwner)
	})

	t.Run("ลดที่นั่งต่ำกว่าคนสูงสุดของการจองที่จะถึง → SeatsBelowBookingsError และไม่บันทึก", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(existing, nil)
		repo.EXPECT().FutureBookings(ctx, id, fixedNow).Return(booked, nil)
		in := baseInput()
		in.Seats = 6
		_, err := newServiceWith(repo).Update(ctx, owner, id, in)
		var e *booking.SeatsBelowBookingsError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, 7, e.Peak)
	})

	t.Run("ย่นเวลาปิดเป็น 21:00 → HoursConflictError", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(existing, nil)
		repo.EXPECT().FutureBookings(ctx, id, fixedNow).Return(booked, nil)
		in := baseInput()
		in.CloseMinute = 21 * 60
		_, err := newServiceWith(repo).Update(ctx, owner, id, in)
		var e *booking.HoursConflictError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, []uuid.UUID{booked[1].ID}, e.BookingIDs)
	})

	t.Run("ลดที่นั่งเท่ากับ peak พอดี → บันทึกได้", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(existing, nil)
		repo.EXPECT().FutureBookings(ctx, id, fixedNow).Return(booked, nil)
		repo.EXPECT().Update(ctx, mock.MatchedBy(func(r *Restaurant) bool { return r.Seats == 7 })).Return(nil)
		repo.EXPECT().FindByID(ctx, id).Return(existing, nil)
		in := baseInput()
		in.Seats = 7
		_, err := newServiceWith(repo).Update(ctx, owner, id, in)
		assert.NoError(t, err)
	})

	t.Run("แก้แค่ชื่อ (ไม่ลดที่นั่ง ไม่เปลี่ยนเวลา) → ไม่ต้องอ่านการจอง", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(existing, nil)
		repo.EXPECT().Update(ctx, mock.Anything).Return(nil)
		repo.EXPECT().FindByID(ctx, id).Return(existing, nil)
		_, err := newServiceWith(repo).Update(ctx, owner, id, baseInput())
		assert.NoError(t, err)
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	owner, id := uuid.New(), uuid.New()

	t.Run("ยกเลิกการจองที่ยังไม่เริ่มแล้ว soft delete ในทรานแซกชันเดียวกัน", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(Restaurant{ID: id, OwnerID: owner}, nil)
		customer, bid := uuid.New(), uuid.New()
		repo.EXPECT().UpcomingBookings(ctx, id, fixedNow).Return([]AffectedBooking{
			{Booking: booking.Booking{ID: bid, UserID: customer, StartAt: time.Date(2026, 10, 10, 19, 0, 0, 0, booking.Bangkok)}},
		}, nil)
		repo.EXPECT().CancelByRestaurant(ctx, []uuid.UUID{bid}, "ร้านปิดให้บริการ", fixedNow).Return(nil)
		repo.EXPECT().Notify(ctx, notification.Draft{Recipient: customer, Kind: notification.KindCancelledByRestaurant,
			BookingID: bid, BusinessDate: "2026-10-10", Reason: "ร้านปิดให้บริการ"}).Return(nil)
		repo.EXPECT().SoftDelete(ctx, id).Return(nil)
		assert.NoError(t, newServiceWith(repo).Delete(ctx, owner, id))
	})

	t.Run("ไม่ใช่เจ้าของ → ไม่แตะการจอง", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(Restaurant{ID: id, OwnerID: owner}, nil)
		assert.ErrorIs(t, newServiceWith(repo).Delete(ctx, uuid.New(), id), ErrNotOwner)
	})
}

func TestDeleteImage(t *testing.T) {
	ctx := context.Background()
	owner, id, img := uuid.New(), uuid.New(), uuid.New()

	t.Run("รูปสุดท้าย → ErrImageRequired", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(Restaurant{ID: id, OwnerID: owner}, nil)
		repo.EXPECT().CountImages(ctx, id).Return(1, nil)
		assert.ErrorIs(t, newServiceWith(repo).DeleteImage(ctx, owner, id, img), ErrImageRequired)
	})

	t.Run("มีหลายรูป → ลบได้", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(Restaurant{ID: id, OwnerID: owner}, nil)
		repo.EXPECT().CountImages(ctx, id).Return(2, nil)
		repo.EXPECT().DeleteImage(ctx, id, img).Return(nil)
		assert.NoError(t, newServiceWith(repo).DeleteImage(ctx, owner, id, img))
	})
}

func TestNextAvailable(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	rest := Restaurant{ID: id, Seats: 4, OpenMinute: 18 * 60, CloseMinute: 22 * 60}
	date := time.Date(2026, 10, 10, 0, 0, 0, 0, booking.Bangkok)

	t.Run("วันที่ 11 เต็มรอบ 19:00 → คืนวันที่ 12", func(t *testing.T) {
		full := []booking.Booking{{PartySize: 4, StartAt: time.Date(2026, 10, 11, 18, 0, 0, 0, booking.Bangkok), EndAt: time.Date(2026, 10, 11, 22, 0, 0, 0, booking.Bangkok), Status: booking.StatusActive}}
		repo := NewMockRepository(t)
		repo.EXPECT().FindByID(ctx, id).Return(rest, nil)
		repo.EXPECT().BookingsBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(full, nil)
		repo.EXPECT().ClosuresBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)
		next, err := newServiceWith(repo).NextAvailable(ctx, id, date, 19*60, 2)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, "2026-10-12", next.Date.Format("2006-01-02"))
		assert.NotEmpty(t, next.Slots)
	})

	t.Run("ค้นเวลาที่ร้านยังไม่เปิด (10:30 ที่ร้าน 18:00–22:00) → ช่วงว่างแรกของรอบวันเดียวกัน", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindByID(ctx, id).Return(rest, nil)
		repo.EXPECT().BookingsBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)
		repo.EXPECT().ClosuresBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)
		next, err := newServiceWith(repo).NextAvailable(ctx, id, date, 10*60+30, 2)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, "2026-10-10", next.Date.Format("2006-01-02"))
		require.Len(t, next.Slots, 5)
		assert.True(t, next.Slots[0].StartAt.Equal(time.Date(2026, 10, 10, 18, 0, 0, 0, booking.Bangkok)))
	})

	t.Run("จำนวนคนมากกว่าที่นั่งทั้งร้าน → ไม่เจอเลย คืน nil", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindByID(ctx, id).Return(rest, nil)
		repo.EXPECT().BookingsBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)
		repo.EXPECT().ClosuresBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)
		next, err := newServiceWith(repo).NextAvailable(ctx, id, date, 19*60, 5)
		require.NoError(t, err)
		assert.Nil(t, next)
	})
}

func TestParseClock(t *testing.T) {
	m, err := ParseClock("18:30")
	require.NoError(t, err)
	assert.Equal(t, 18*60+30, m)
	_, err = ParseClock("18:15")
	assert.Error(t, err, "ต้องลง :00/:30")
	_, err = ParseClock("25:00")
	assert.Error(t, err)
}

func TestCreateClosure(t *testing.T) {
	ctx := context.Background()
	owner, id, customer := uuid.New(), uuid.New(), uuid.New()
	rest := Restaurant{ID: id, OwnerID: owner, Seats: 10, OpenMinute: 11 * 60, CloseMinute: 22 * 60}
	bkk := func(h, m int) time.Time { return time.Date(2026, 10, 10, h, m, 0, 0, booking.Bangkok) }
	// fixedNow = 10 ต.ค. 09:00 → ปิด 10 ต.ค. 18:00–20:00
	in := ClosureInput{Partial: true, Date: bkk(0, 0), StartMinute: 18 * 60, EndMinute: 20 * 60, Reason: "ไฟดับ"}
	start, end := bkk(18, 0), bkk(20, 0)
	hit := AffectedBooking{Booking: booking.Booking{ID: uuid.New(), UserID: customer, StartAt: bkk(19, 0), EndAt: bkk(20, 0)}, CustomerName: "มะลิ"}
	isClosure := mock.MatchedBy(func(c *booking.Closure) bool {
		return c.RestaurantID == id && c.StartAt.Equal(start) && c.EndAt.Equal(end) && c.Reason == "ไฟดับ"
	})

	t.Run("ไม่มีการจองทับ → บันทึกเลย", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return(nil, nil)
		repo.EXPECT().CreateClosure(ctx, isClosure).Return(nil)
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, in)
		assert.NoError(t, err)
	})

	t.Run("มีการจองทับแต่ยังไม่ยืนยัน → AffectsBookingsError พร้อมรายการ ไม่บันทึกอะไร", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{hit}, nil)
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, in)
		var affects *AffectsBookingsError
		require.True(t, errors.As(err, &affects))
		assert.Equal(t, []AffectedBooking{hit}, affects.Bookings)
	})

	t.Run("ยืนยันตรงรายการ → บันทึก + ยกเลิก + แจ้งลูกค้า", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{hit}, nil)
		repo.EXPECT().CreateClosure(ctx, isClosure).Return(nil)
		repo.EXPECT().CancelByRestaurant(ctx, []uuid.UUID{hit.ID}, "ไฟดับ", fixedNow).Return(nil)
		repo.EXPECT().Notify(ctx, notification.Draft{Recipient: customer, Kind: notification.KindCancelledByRestaurant,
			BookingID: hit.ID, BusinessDate: "2026-10-10", Reason: "ไฟดับ"}).Return(nil)
		confirmed := in
		confirmed.ConfirmBookingIDs = []uuid.UUID{hit.ID}
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, confirmed)
		assert.NoError(t, err)
	})

	t.Run("ยืนยันรายการเก่า แต่มีการจองใหม่แทรก → 409 ใหม่ ไม่ยกเลิกอะไร", func(t *testing.T) {
		newcomer := AffectedBooking{Booking: booking.Booking{ID: uuid.New(), UserID: uuid.New(), StartAt: bkk(18, 30), EndAt: bkk(19, 30)}}
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{newcomer, hit}, nil)
		confirmed := in
		confirmed.ConfirmBookingIDs = []uuid.UUID{hit.ID}
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, confirmed)
		var affects *AffectsBookingsError
		require.True(t, errors.As(err, &affects))
		assert.Len(t, affects.Bookings, 2)
	})

	t.Run("การจองของเจ้าของร้านเอง → ยกเลิก แต่ไม่แจ้งตัวเอง", func(t *testing.T) {
		mine := AffectedBooking{Booking: booking.Booking{ID: uuid.New(), UserID: owner, StartAt: bkk(19, 0), EndAt: bkk(20, 0)}}
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		repo.EXPECT().AffectedBookings(ctx, id, start, end, fixedNow).Return([]AffectedBooking{mine}, nil)
		repo.EXPECT().CreateClosure(ctx, isClosure).Return(nil)
		repo.EXPECT().CancelByRestaurant(ctx, []uuid.UUID{mine.ID}, "ไฟดับ", fixedNow).Return(nil)
		confirmed := in
		confirmed.ConfirmBookingIDs = []uuid.UUID{mine.ID}
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, confirmed)
		assert.NoError(t, err)
	})

	t.Run("เริ่มก่อนตอนนี้ (08:00 ขณะที่ตอนนี้ 09:00) → ErrInvalidClosure", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		past := in
		past.StartMinute = 8 * 60
		_, err := newServiceWith(repo).CreateClosure(ctx, owner, id, past)
		assert.ErrorIs(t, err, ErrInvalidClosure)
	})

	t.Run("ไม่ใช่เจ้าของ → ErrNotOwner", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, id).Return(rest, nil)
		_, err := newServiceWith(repo).CreateClosure(ctx, uuid.New(), id, in)
		assert.ErrorIs(t, err, ErrNotOwner)
	})
}
