package notification

import (
	"context"
	"testing"
	"time"

	"jongyoung/internal/platform/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// เวลาไทย (ไม่มี DST) — ไม่ import booking เพราะ booking import notification (import cycle)
var bangkokTest = time.FixedZone("Asia/Bangkok", 7*60*60)

func at(h int) time.Time { return time.Date(2026, 10, 10, h, 0, 0, 0, bangkokTest) }

func insertUser(t *testing.T, db *gorm.DB, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO users (keycloak_uid, email, display_name) VALUES (?, 'u@x.y', ?) RETURNING id`, uuid.NewString(), name).Row().Scan(&id))
	return id
}

func insertBooking(t *testing.T, db *gorm.DB, owner, customer uuid.UUID, party int) uuid.UUID {
	t.Helper()
	var rid, bid uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO restaurants (owner_id, name, address, seats, open_minute, close_minute)
		VALUES (?, 'ครัวบ้านสวน', 'กรุงเทพฯ', 10, 660, 1320) RETURNING id`, owner).Row().Scan(&rid))
	require.NoError(t, db.Raw(`INSERT INTO bookings (restaurant_id, user_id, party_size, start_at, end_at)
		VALUES (?, ?, ?, ?, ?) RETURNING id`, rid, customer, party, at(18), at(19)).Row().Scan(&bid))
	return bid
}

func TestInsertSnapshot(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	owner, customer := insertUser(t, db, "เจ้าของ"), insertUser(t, db, "มะลิ")
	bid := insertBooking(t, db, owner, customer, 4)

	require.NoError(t, Insert(ctx, db, Draft{Recipient: owner, Kind: KindBookingCreated, BookingID: bid, BusinessDate: "2026-10-10"}))
	require.NoError(t, db.Exec(`UPDATE bookings SET party_size = 2 WHERE id = ?`, bid).Error)

	var n Notification
	require.NoError(t, db.First(&n, "user_id = ?", owner).Error)
	assert.Equal(t, KindBookingCreated, n.Kind)
	assert.Equal(t, "ครัวบ้านสวน", n.RestaurantName)
	assert.Equal(t, "มะลิ", n.CustomerName)
	assert.Equal(t, 4, n.PartySize, "snapshot ตอนเกิดเหตุ — แก้การจองทีหลังไม่กระทบ")
	assert.Equal(t, "2026-10-10", n.BusinessDate.Format("2006-01-02"))
	assert.True(t, n.StartAt.Equal(at(18)))
	assert.Nil(t, n.ReadAt)
}
