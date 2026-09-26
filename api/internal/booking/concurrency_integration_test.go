package booking

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"jongyoung/internal/platform/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// เทสต์นี้ใช้ Postgres จริง เพื่อพิสูจน์ว่า FOR UPDATE กันการจองชนกันได้จริง ไม่ใช่แค่ใน mock

func insertUser(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO users (keycloak_uid, email, display_name) VALUES (?, 'u@x.y', 'u') RETURNING id`, uuid.NewString()).Row().Scan(&id))
	return id
}

func insertRestaurant(t *testing.T, db *gorm.DB, owner uuid.UUID, seats int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO restaurants (owner_id, name, address, seats, open_minute, close_minute)
		VALUES (?, 'ร้านทดสอบ', 'กรุงเทพฯ', ?, 660, 1320) RETURNING id`, owner, seats).Row().Scan(&id))
	return id
}

// runTogether ยิง fn พร้อมกันหลาย goroutine แล้วคืน error ของแต่ละตัว
func runTogether(n int, fn func(i int) error) []error {
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // ปล่อยทุกตัวพร้อมกัน
			errs[i] = fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
	return errs
}

func TestConcurrentBooking(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := func() time.Time { return bkk(2026, 10, 1, 12, 0) }
	svc := NewService(NewRepository(db), now)
	choice := Choice{Date: bkk(2026, 10, 10, 0, 0), StartMinute: 19 * 60, EndMinute: 20 * 60, PartySize: 6}

	t.Run("25) ร้านเหลือ 6 ที่ สองคนกดคนละ 6 พร้อมกัน → สำเร็จ 1 ได้ NOT_ENOUGH_SEATS 1", func(t *testing.T) {
		owner := insertUser(t, db)
		rid := insertRestaurant(t, db, owner, 6)
		users := []uuid.UUID{insertUser(t, db), insertUser(t, db)}

		errs := runTogether(2, func(i int) error {
			_, err := svc.Create(ctx, users[i], rid, choice)
			return err
		})

		ok, full := 0, 0
		for _, err := range errs {
			var nes *NotEnoughSeatsError
			switch {
			case err == nil:
				ok++
			case errors.As(err, &nes):
				full++
			default:
				t.Fatalf("error ที่ไม่คาดคิด: %v", err)
			}
		}
		assert.Equal(t, 1, ok)
		assert.Equal(t, 1, full)
	})

	t.Run("26) คนเดียวกดซ้ำพร้อมกัน → สำเร็จ 1 ได้ DUPLICATE_BOOKING 1 (ไม่ใช่ NOT_ENOUGH_SEATS)", func(t *testing.T) {
		owner := insertUser(t, db)
		rid := insertRestaurant(t, db, owner, 20) // ที่นั่งเหลือเฟือ — ถ้าได้ error ต้องเป็นเพราะจองซ้อนเท่านั้น
		me := insertUser(t, db)

		errs := runTogether(2, func(int) error {
			_, err := svc.Create(ctx, me, rid, choice)
			return err
		})

		ok, dup := 0, 0
		for _, err := range errs {
			var d *DuplicateBookingError
			switch {
			case err == nil:
				ok++
			case errors.As(err, &d):
				dup++
			default:
				t.Fatalf("error ที่ไม่คาดคิด: %v", err)
			}
		}
		assert.Equal(t, 1, ok)
		assert.Equal(t, 1, dup)

		var count int64
		require.NoError(t, db.Model(&Booking{}).Where("restaurant_id = ? AND user_id = ?", rid, me).Count(&count).Error)
		assert.EqualValues(t, 1, count, "ต้องมีการจองในฐานข้อมูลแค่ 1 รายการ")
	})

	t.Run("ยกเลิกแล้วที่นั่งกลับมาให้คนอื่นจองได้", func(t *testing.T) {
		owner := insertUser(t, db)
		rid := insertRestaurant(t, db, owner, 6)
		a, b := insertUser(t, db), insertUser(t, db)
		first, err := svc.Create(ctx, a, rid, choice)
		require.NoError(t, err)
		_, err = svc.Create(ctx, b, rid, choice)
		require.Error(t, err)
		require.NoError(t, svc.Cancel(ctx, a, first.ID))
		_, err = svc.Create(ctx, b, rid, choice)
		assert.NoError(t, err)
	})

	t.Run("ListByUser และ FindView ดึงชื่อร้าน/ลูกค้ามาด้วย", func(t *testing.T) {
		owner := insertUser(t, db)
		rid := insertRestaurant(t, db, owner, 10)
		me := insertUser(t, db)
		created, err := svc.Create(ctx, me, rid, choice)
		require.NoError(t, err)

		list, err := svc.ListMine(ctx, me, "upcoming")
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, "ร้านทดสอบ", list[0].RestaurantName)
		assert.Equal(t, owner, list[0].RestaurantOwnerID)

		v, err := svc.Get(ctx, owner, created.ID)
		require.NoError(t, err, "เจ้าของร้านดูการจองในร้านตัวเองได้")
		assert.Equal(t, "u", v.CustomerName)
	})
}

// ช่วงพักต้องถูกอ่านจาก DB ตอนจองด้วย (restaurantColumns) — ไม่งั้นจองช่วงพักผ่าน API ได้
func TestBookingInBreakRejected(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := func() time.Time { return bkk(2026, 10, 1, 12, 0) }
	svc := NewService(NewRepository(db), now)
	rid := insertRestaurant(t, db, insertUser(t, db), 10) // 11:00–22:00
	require.NoError(t, db.Exec(`UPDATE restaurants SET break_start_minute = 900, break_end_minute = 1020 WHERE id = ?`, rid).Error)

	inBreak := Choice{Date: bkk(2026, 10, 10, 0, 0), StartMinute: 15 * 60, EndMinute: 16 * 60, PartySize: 2}
	_, err := svc.Create(ctx, insertUser(t, db), rid, inBreak)
	assert.ErrorIs(t, err, ErrOutsideHours)

	afterBreak := Choice{Date: bkk(2026, 10, 10, 0, 0), StartMinute: 17 * 60, EndMinute: 18 * 60, PartySize: 2}
	_, err = svc.Create(ctx, insertUser(t, db), rid, afterBreak)
	assert.NoError(t, err)
}

// ช่วงปิดชั่วคราวต้องถูกตรวจในล็อกเดียวกับการจอง และจองสำเร็จต้องมีแจ้งเตือนถึงเจ้าของร้านในทรานแซกชันเดียวกัน
func TestBookingClosureAndNotify(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := func() time.Time { return bkk(2026, 10, 1, 12, 0) }
	svc := NewService(NewRepository(db), now)
	owner := insertUser(t, db)
	rid := insertRestaurant(t, db, owner, 10) // 11:00–22:00
	require.NoError(t, db.Exec(`INSERT INTO restaurant_closures (restaurant_id, start_at, end_at, reason) VALUES (?, ?, ?, 'ไฟดับ')`,
		rid, bkk(2026, 10, 10, 18, 0), bkk(2026, 10, 10, 20, 0)).Error)
	date := bkk(2026, 10, 10, 0, 0)

	_, err := svc.Create(ctx, insertUser(t, db), rid, Choice{Date: date, StartMinute: 19 * 60, EndMinute: 20 * 60, PartySize: 2})
	var closed *ClosedError
	require.True(t, errors.As(err, &closed), "err = %v", err)
	assert.Equal(t, "ไฟดับ", closed.Closure.Reason)

	b, err := svc.Create(ctx, insertUser(t, db), rid, Choice{Date: date, StartMinute: 20 * 60, EndMinute: 21 * 60, PartySize: 2})
	require.NoError(t, err)
	var n int64
	require.NoError(t, db.Table("notifications").Where("user_id = ? AND booking_id = ? AND kind = 'booking_created'", owner, b.ID).Count(&n).Error)
	assert.EqualValues(t, 1, n)
}
