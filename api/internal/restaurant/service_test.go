package restaurant

import (
	"context"
	"errors"
	"testing"
	"time"

	"jongyoung/internal/booking"

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
		repo.EXPECT().CancelFutureBookings(ctx, id, fixedNow).Return(3, nil)
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
		next, err := newServiceWith(repo).NextAvailable(ctx, id, date, 19*60, 2)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, "2026-10-12", next.Date.Format("2006-01-02"))
		assert.NotEmpty(t, next.Slots)
	})

	t.Run("จำนวนคนมากกว่าที่นั่งทั้งร้าน → ไม่เจอเลย คืน nil", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindByID(ctx, id).Return(rest, nil)
		repo.EXPECT().BookingsBetween(ctx, []uuid.UUID{id}, mock.Anything, mock.Anything).Return(nil, nil)
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
