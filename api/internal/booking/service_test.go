package booking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var svcNow = bkk(2026, 10, 10, 12, 0)

func newSvc(repo *MockRepository) *service {
	return NewService(repo, func() time.Time { return svcNow })
}

func expectTx(repo *MockRepository) {
	repo.EXPECT().Transaction(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(Repository) error) error { return fn(repo) })
}

func TestServiceCreate(t *testing.T) {
	ctx := context.Background()
	user, rid := uuid.New(), uuid.New()
	rest := RestaurantInfo{ID: rid, Seats: 10, OpenMinute: 18 * 60, CloseMinute: 2 * 60, CancelBeforeMinutes: 30}
	date := bkk(2026, 10, 10, 0, 0)
	// เลือกวันทำการ 10 เวลา 23:30–00:30 → ต้องได้ 23:30 วันที่ 10 ถึง 00:30 วันที่ 11
	choice := Choice{Date: date, StartMinute: 23*60 + 30, EndMinute: 30, PartySize: 2}
	start, end := bkk(2026, 10, 10, 23, 30), bkk(2026, 10, 11, 0, 30)

	t.Run("ข้ามเที่ยงคืน: แปลงเวลาถูก แล้วบันทึก", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().FindOverlappingOwn(ctx, rid, user, start, end, uuid.Nil).Return(nil, nil)
		repo.EXPECT().Overlapping(ctx, rid, start, end, uuid.Nil).Return(nil, nil)
		repo.EXPECT().Create(ctx, mock.MatchedBy(func(b *Booking) bool {
			return b.StartAt.Equal(start) && b.EndAt.Equal(end) && b.UserID == user && b.Status == StatusActive
		})).Return(nil)
		_, err := newSvc(repo).Create(ctx, user, rid, choice)
		assert.NoError(t, err)
	})

	t.Run("กฎ 8: มีการจองของตัวเองทับอยู่ → DuplicateBookingError และไม่นับที่นั่ง", func(t *testing.T) {
		existingID := uuid.New()
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().FindOverlappingOwn(ctx, rid, user, start, end, uuid.Nil).Return(&Booking{ID: existingID}, nil)
		_, err := newSvc(repo).Create(ctx, user, rid, choice)
		var dup *DuplicateBookingError
		require.True(t, errors.As(err, &dup))
		assert.Equal(t, existingID, dup.BookingID)
	})

	t.Run("กฎ 7: ที่นั่งไม่พอ → NotEnoughSeatsError และไม่บันทึก", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().FindOverlappingOwn(ctx, rid, user, start, end, uuid.Nil).Return(nil, nil)
		repo.EXPECT().Overlapping(ctx, rid, start, end, uuid.Nil).Return([]Booking{bk(9, start, end)}, nil)
		_, err := newSvc(repo).Create(ctx, user, rid, choice)
		var nes *NotEnoughSeatsError
		require.True(t, errors.As(err, &nes))
		assert.Equal(t, 1, nes.Available)
	})

	t.Run("กฎที่ไม่ใช้ DB ผิด → RuleError ก่อนแตะ booking อื่น", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		bad := choice
		bad.PartySize = 11
		_, err := newSvc(repo).Create(ctx, user, rid, bad)
		assert.ErrorIs(t, err, ErrPartyTooLarge)
	})

	t.Run("ร้านไม่มี/ถูกลบ → ErrRestaurantNotFound", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(RestaurantInfo{}, ErrRestaurantNotFound)
		_, err := newSvc(repo).Create(ctx, user, rid, choice)
		assert.ErrorIs(t, err, ErrRestaurantNotFound)
	})
}

func TestServiceUpdateAndCancel(t *testing.T) {
	ctx := context.Background()
	user, rid, bid := uuid.New(), uuid.New(), uuid.New()
	rest := RestaurantInfo{ID: rid, Seats: 10, OpenMinute: 11 * 60, CloseMinute: 22 * 60, CancelBeforeMinutes: 30}
	mine := Booking{ID: bid, RestaurantID: rid, UserID: user, PartySize: 7,
		StartAt: bkk(2026, 10, 10, 19, 0), EndAt: bkk(2026, 10, 10, 20, 0), Status: StatusActive}
	date := bkk(2026, 10, 10, 0, 0)

	t.Run("3) A แก้เป็น 8 คน ขณะ B จอง 3 → ปฏิเสธ (excludeID = booking นี้)", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindByID(ctx, bid).Return(mine, nil)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().LockByID(ctx, bid).Return(mine, nil)
		repo.EXPECT().FindOverlappingOwn(ctx, rid, user, mine.StartAt, mine.EndAt, bid).Return(nil, nil)
		repo.EXPECT().Overlapping(ctx, rid, mine.StartAt, mine.EndAt, bid).Return([]Booking{bk(3, mine.StartAt, mine.EndAt)}, nil)
		_, err := newSvc(repo).Update(ctx, user, bid, Choice{Date: date, StartMinute: 19 * 60, EndMinute: 20 * 60, PartySize: 8})
		var nes *NotEnoughSeatsError
		assert.True(t, errors.As(err, &nes))
	})

	t.Run("แก้การจองของคนอื่น → ErrForbidden", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindByID(ctx, bid).Return(mine, nil)
		expectTx(repo)
		repo.EXPECT().LockRestaurant(ctx, rid).Return(rest, nil)
		repo.EXPECT().LockByID(ctx, bid).Return(mine, nil)
		_, err := newSvc(repo).Update(ctx, uuid.New(), bid, Choice{Date: date, StartMinute: 19 * 60, EndMinute: 20 * 60, PartySize: 2})
		assert.ErrorIs(t, err, ErrForbidden)
	})

	// cancelWith เตรียม mock ของการยกเลิก booking b ณ เวลา now แล้วคืน error ที่ได้
	cancelWith := func(t *testing.T, b Booking, now time.Time) error {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, bid).Return(b, nil)
		repo.EXPECT().FindRestaurant(ctx, rid, true).Return(rest, nil)
		return NewService(repo, func() time.Time { return now }).Cancel(ctx, user, bid)
	}

	t.Run("Cancel: ยกเลิกไปแล้ว → ErrBookingCancelled", func(t *testing.T) {
		cancelled := mine
		cancelled.Status = StatusCancelled
		assert.ErrorIs(t, cancelWith(t, cancelled, svcNow), ErrBookingCancelled)
	})

	t.Run("Cancel: เริ่มไปแล้ว → ErrAlreadyStarted", func(t *testing.T) {
		assert.ErrorIs(t, cancelWith(t, mine, bkk(2026, 10, 10, 19, 10)), ErrAlreadyStarted)
	})

	t.Run("Cancel: เลยเส้นตาย 18:30 → CancelWindowError บอกเวลาที่ยกเลิกได้ถึง", func(t *testing.T) {
		var w *CancelWindowError
		require.True(t, errors.As(cancelWith(t, mine, bkk(2026, 10, 10, 18, 31)), &w))
		assert.True(t, w.Until.Equal(bkk(2026, 10, 10, 18, 30)))
	})

	t.Run("Cancel: ตรงเส้นตายพอดี → ยกเลิกได้", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().LockByID(ctx, bid).Return(mine, nil)
		repo.EXPECT().FindRestaurant(ctx, rid, true).Return(rest, nil)
		deadline := bkk(2026, 10, 10, 18, 30)
		repo.EXPECT().Cancel(ctx, bid, deadline).Return(nil)
		assert.NoError(t, NewService(repo, func() time.Time { return deadline }).Cancel(ctx, user, bid))
	})
}

func TestServiceGetAndBoard(t *testing.T) {
	ctx := context.Background()
	customer, owner, stranger, bid, rid := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	v := View{Booking: Booking{ID: bid, UserID: customer}, RestaurantOwnerID: owner}

	for _, who := range []uuid.UUID{customer, owner} {
		repo := NewMockRepository(t)
		repo.EXPECT().FindView(ctx, bid).Return(v, nil)
		_, err := newSvc(repo).Get(ctx, who, bid)
		assert.NoError(t, err, "เจ้าของการจองและเจ้าของร้านดูได้")
	}
	repo := NewMockRepository(t)
	repo.EXPECT().FindView(ctx, bid).Return(v, nil)
	_, err := newSvc(repo).Get(ctx, stranger, bid)
	assert.ErrorIs(t, err, ErrForbidden)

	t.Run("20) บอร์ดวันเสาร์ของร้านข้ามคืนใช้รอบ 18:00 เสาร์ – 02:00 อาทิตย์", func(t *testing.T) {
		rest := RestaurantInfo{ID: rid, OwnerID: owner, Seats: 10, OpenMinute: 18 * 60, CloseMinute: 2 * 60}
		repo := NewMockRepository(t)
		repo.EXPECT().FindRestaurant(ctx, rid, false).Return(rest, nil)
		repo.EXPECT().ListForRestaurant(ctx, rid, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 11, 2, 0)).Return([]View{
			{Booking: bk(4, bkk(2026, 10, 11, 1, 0), bkk(2026, 10, 11, 2, 0))},
		}, nil)
		board, err := newSvc(repo).Board(ctx, owner, rid, bkk(2026, 10, 10, 0, 0))
		require.NoError(t, err)
		assert.Len(t, board.Slots, 16, "ไม่ตัดช่วงที่ผ่านไปแล้วออก")
		assert.Equal(t, 6, board.Slots[14].Available, "01:00 มีคน 4")
	})

	t.Run("คนอื่นดูบอร์ด → ErrNotRestaurantOwner", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindRestaurant(ctx, rid, false).Return(RestaurantInfo{ID: rid, OwnerID: owner}, nil)
		_, err := newSvc(repo).Board(ctx, stranger, rid, bkk(2026, 10, 10, 0, 0))
		assert.ErrorIs(t, err, ErrNotRestaurantOwner)
	})
}

func TestBusinessDateLabel(t *testing.T) {
	assert.Equal(t, "2026-10-10", businessDate(bkk(2026, 10, 11, 0, 30), overnight), "00:30 เช้าวันที่ 11 เป็นของรอบวันที่ 10")
	assert.Equal(t, "2026-10-10", businessDate(bkk(2026, 10, 10, 23, 0), overnight))
	assert.Equal(t, "2026-10-10", businessDate(bkk(2026, 10, 10, 12, 0), normal))
	assert.Equal(t, "2026-10-11", businessDate(bkk(2026, 10, 11, 0, 30), allDay))
}

func TestCode(t *testing.T) {
	assert.Equal(t, "JY-3FA85F", Code(uuid.MustParse("3fa85f64-5717-4562-b3fc-2c963f66afa6")))
}
