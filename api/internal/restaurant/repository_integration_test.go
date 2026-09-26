package restaurant

import (
	"context"
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

	t.Run("CancelFutureBookings ยกเลิกเฉพาะที่ยังไม่เริ่ม", func(t *testing.T) {
		n, err := repo.CancelFutureBookings(ctx, r.ID, now)
		require.NoError(t, err)
		assert.EqualValues(t, 1, n)
		var status string
		require.NoError(t, db.Raw("SELECT status FROM bookings WHERE id = ?", past).Scan(&status).Error)
		assert.Equal(t, booking.StatusActive, status, "การจองที่ผ่านไปแล้วต้องไม่ถูกยกเลิก (เก็บประวัติ)")
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
