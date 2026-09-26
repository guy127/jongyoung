package restaurant

import (
	"context"
	"errors"
	"testing"
	"time"

	"jongyoung/internal/booking"
	"jongyoung/internal/platform/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createOwner(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO users (keycloak_uid, email, display_name) VALUES (?, 'o@x.y', 'owner') RETURNING id`, uuid.NewString()).Row().Scan(&id))
	return id
}

func createRestaurant(t *testing.T, repo *repository, owner uuid.UUID, name string, ratingSum, ratingCount int) Restaurant {
	t.Helper()
	r := Restaurant{OwnerID: owner, Name: name, Address: "กรุงเทพฯ", Seats: 10, OpenMinute: 660, CloseMinute: 1320,
		CancelBeforeMinutes: 30, RatingSum: ratingSum, RatingCount: ratingCount,
		Images: []Image{{URL: "https://img.test/" + name, SortOrder: 0}}}
	require.NoError(t, repo.Create(context.Background(), &r))
	return r
}

func TestRepositoryList(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()
	owner := createOwner(t, db)

	few := createRestaurant(t, repo, owner, "ห้าดาวรีวิวเดียว", 5, 1)       // ★5.0 จาก 1 รีวิว
	many := createRestaurant(t, repo, owner, "สี่จุดแปดรีวิวเยอะ", 221, 46) // ★4.8 จาก 46 รีวิว
	none := createRestaurant(t, repo, owner, "ยังไม่มีรีวิว", 0, 0)
	low := createRestaurant(t, repo, owner, "คะแนนต่ำ", 60, 20) // ★3.0 จาก 20 รีวิว
	deleted := createRestaurant(t, repo, owner, "ร้านที่ถูกลบ", 50, 10)
	require.NoError(t, repo.SoftDelete(ctx, deleted.ID))

	t.Run("sort=rating: Bayesian — ★5.0 จาก 1 รีวิวไม่แซง ★4.8 จาก 46; ไม่มีรีวิวอยู่ท้าย; ร้านที่ลบไม่แสดง", func(t *testing.T) {
		list, total, err := repo.List(ctx, ListQuery{Sort: "rating", Limit: 10})
		require.NoError(t, err)
		assert.EqualValues(t, 4, total)
		names := []string{}
		for _, r := range list {
			names = append(names, r.Name)
		}
		assert.Equal(t, []string{many.Name, few.Name, low.Name, none.Name}, names)
		assert.Len(t, list[0].Images, 1, "preload รูปมาด้วย")
	})

	t.Run("sort=reviews", func(t *testing.T) {
		list, _, err := repo.List(ctx, ListQuery{Sort: "reviews", Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, many.ID, list[0].ID)
	})

	t.Run("ค้นด้วยชื่อ + pagination", func(t *testing.T) {
		list, total, err := repo.List(ctx, ListQuery{Q: "รีวิว", Limit: 1, Offset: 1})
		require.NoError(t, err)
		assert.EqualValues(t, 3, total)
		assert.Len(t, list, 1)
	})
}

func TestRepositoryBookings(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()
	owner := createOwner(t, db)
	r := createRestaurant(t, repo, owner, "ร้านทดสอบ", 0, 0)

	at := func(h int) time.Time { return time.Date(2026, 10, 10, h, 0, 0, 0, booking.Bangkok) }
	insert := func(start, end time.Time, status string) uuid.UUID {
		b := booking.Booking{RestaurantID: r.ID, UserID: owner, PartySize: 2, StartAt: start, EndAt: end, Status: status}
		require.NoError(t, db.Create(&b).Error)
		return b.ID
	}
	past := insert(at(11), at(12), booking.StatusActive)
	future := insert(at(19), at(20), booking.StatusActive)
	insert(at(19), at(20), booking.StatusCancelled)
	now := at(15)

	t.Run("BookingsBetween ไม่รวมที่ยกเลิก", func(t *testing.T) {
		list, err := repo.BookingsBetween(ctx, []uuid.UUID{r.ID}, at(18), at(21))
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, future, list[0].ID)
	})

	t.Run("UpcomingBookings เฉพาะที่ยังไม่เริ่ม", func(t *testing.T) {
		list, err := repo.UpcomingBookings(ctx, r.ID, now)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, future, list[0].ID)
		assert.NotEqual(t, past, list[0].ID, "การจองที่ผ่านไปแล้วต้องไม่ถูกยกเลิก (เก็บประวัติ)")
	})

	t.Run("AddImage ต่อท้าย sort_order", func(t *testing.T) {
		img := Image{RestaurantID: r.ID, URL: "https://img.test/2"}
		require.NoError(t, repo.AddImage(ctx, &img))
		assert.Equal(t, 1, img.SortOrder)
		n, err := repo.CountImages(ctx, r.ID)
		require.NoError(t, err)
		assert.EqualValues(t, 2, n)
	})
}

func TestRepositoryUpdate(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()
	r := createRestaurant(t, repo, createOwner(t, db), "ร้านแก้ไข", 0, 0)

	r.MapURL = "https://maps.app.goo.gl/AbCdEf123"
	r.ClosedWeekdays = booking.WeekdayMask(time.Monday)
	require.NoError(t, repo.Update(ctx, &r))

	got, err := repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.MapURL, got.MapURL)
	assert.Equal(t, r.ClosedWeekdays, got.ClosedWeekdays, "วันปิดต้องถูกบันทึกตอนแก้ไขด้วย")
}

func TestRepositoryUpdateBreak(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()
	r := createRestaurant(t, repo, createOwner(t, db), "ร้านมีช่วงพัก", 0, 0)

	r.BreakStartMinute, r.BreakEndMinute = 15*60, 17*60
	require.NoError(t, repo.Update(ctx, &r))
	got, err := repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, 15*60, got.BreakStartMinute, "ช่วงพักต้องถูกบันทึกตอนแก้ไข")
	assert.Equal(t, 17*60, got.BreakEndMinute)

	r.BreakStartMinute, r.BreakEndMinute = 0, 0
	require.NoError(t, repo.Update(ctx, &r))
	got, err = repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	assert.False(t, got.Hours().HasBreak(), "ลบช่วงพัก (0/0) ต้องบันทึกค่าศูนย์ได้ด้วย")
}

func TestClosures(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()
	owner, customer := createOwner(t, db), createOwner(t, db)
	r := createRestaurant(t, repo, owner, "ร้านปิดชั่วคราว", 0, 0)
	at := func(h int) time.Time { return time.Date(2026, 10, 10, h, 0, 0, 0, booking.Bangkok) }
	insert := func(start, end time.Time) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, db.Raw(`INSERT INTO bookings (restaurant_id, user_id, party_size, start_at, end_at) VALUES (?, ?, 2, ?, ?) RETURNING id`,
			r.ID, customer, start, end).Row().Scan(&id))
		return id
	}
	insert(at(14), at(16)) // เริ่มไปแล้วตอน 15:00 — ลูกค้านั่งอยู่ในร้าน
	later := insert(at(18), at(19))
	insert(at(20), at(21)) // ไม่ทับช่วงปิด 14:00–20:00
	now := at(15)

	t.Run("AffectedBookings ไม่รวมที่เริ่มแล้ว และไม่รวมที่ไม่ทับ", func(t *testing.T) {
		list, err := repo.AffectedBookings(ctx, r.ID, at(14), at(20), now)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, later, list[0].ID)
		assert.Equal(t, "owner", list[0].CustomerName)
	})

	t.Run("CRUD ช่วงปิด + ลบของร้านอื่นไม่ได้", func(t *testing.T) {
		c := booking.Closure{RestaurantID: r.ID, StartAt: at(18), EndAt: at(20), Reason: "ไฟดับ"}
		require.NoError(t, repo.CreateClosure(ctx, &c))
		list, err := repo.ClosuresBetween(ctx, []uuid.UUID{r.ID}, at(19), at(21))
		require.NoError(t, err)
		require.Len(t, list, 1)
		upcoming, err := repo.UpcomingClosures(ctx, r.ID, now)
		require.NoError(t, err)
		require.Len(t, upcoming, 1)
		ok, err := repo.DeleteClosure(ctx, uuid.New(), c.ID)
		require.NoError(t, err)
		assert.False(t, ok, "id ร้านไม่ตรง → ลบไม่ได้")
		ok, err = repo.DeleteClosure(ctx, r.ID, c.ID)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("service: ปิดร้าน → 409 → ยืนยัน → ช่วงปิด + ยกเลิก + แจ้งลูกค้า ในทรานแซกชันเดียว", func(t *testing.T) {
		svc := NewService(repo, func() time.Time { return now })
		in := ClosureInput{Partial: true, Date: at(0), StartMinute: 18 * 60, EndMinute: 20 * 60, Reason: "ไฟดับ"}
		_, err := svc.CreateClosure(ctx, owner, r.ID, in)
		var affects *AffectsBookingsError
		require.True(t, errors.As(err, &affects), "err = %v", err)
		require.Len(t, affects.Bookings, 1)

		in.ConfirmBookingIDs = []uuid.UUID{affects.Bookings[0].ID}
		_, err = svc.CreateClosure(ctx, owner, r.ID, in)
		require.NoError(t, err)

		var got booking.Booking
		require.NoError(t, db.First(&got, "id = ?", later).Error)
		assert.Equal(t, booking.StatusCancelled, got.Status)
		require.NotNil(t, got.CancelledBy)
		assert.Equal(t, booking.CancelledByRestaurant, *got.CancelledBy)
		assert.Equal(t, "ไฟดับ", got.CancelReason)
		var n int64
		require.NoError(t, db.Table("notifications").Where("user_id = ? AND booking_id = ? AND kind = 'booking_cancelled_by_restaurant'", customer, later).Count(&n).Error)
		assert.EqualValues(t, 1, n)
	})
}
