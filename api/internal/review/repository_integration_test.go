package review

import (
	"context"
	"sync"
	"testing"

	"jongyoung/internal/platform/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUser(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO users (keycloak_uid, email, display_name) VALUES (?, 'u@x.y', 'ผู้รีวิว') RETURNING id`, uuid.NewString()).Row().Scan(&id))
	return id
}

func rating(t *testing.T, db *gorm.DB, rid uuid.UUID) (sum, count int) {
	t.Helper()
	require.NoError(t, db.Raw("SELECT rating_sum, rating_count FROM restaurants WHERE id = ?", rid).Row().Scan(&sum, &count))
	return sum, count
}

func TestRatingStaysConsistent(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	svc := NewService(NewRepository(db))
	owner := newUser(t, db)
	var rid uuid.UUID
	require.NoError(t, db.Raw(`INSERT INTO restaurants (owner_id, name, address, seats, open_minute, close_minute)
		VALUES (?, 'ร้าน', 'ที่อยู่', 10, 660, 1320) RETURNING id`, owner).Row().Scan(&rid))

	t.Run("10 คนรีวิวพร้อมกัน → ผลรวมและจำนวนต้องครบ ไม่หาย (atomic update)", func(t *testing.T) {
		var wg sync.WaitGroup
		for range 10 {
			u := newUser(t, db)
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := svc.Create(ctx, u, rid, 4, "ok")
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
		sum, count := rating(t, db, rid)
		assert.Equal(t, 40, sum)
		assert.Equal(t, 10, count)
	})

	t.Run("แก้แล้วลบ → คะแนนรวมกลับมาถูกต้อง", func(t *testing.T) {
		u := newUser(t, db)
		_, err := svc.Create(ctx, u, rid, 1, "แย่")
		require.NoError(t, err)
		_, err = svc.Update(ctx, u, rid, 5, "ดีขึ้น")
		require.NoError(t, err)
		sum, count := rating(t, db, rid)
		assert.Equal(t, 45, sum)
		assert.Equal(t, 11, count)

		require.NoError(t, svc.Delete(ctx, u, rid))
		sum, count = rating(t, db, rid)
		assert.Equal(t, 40, sum)
		assert.Equal(t, 10, count)
	})

	t.Run("List ใส่ชื่อผู้เขียนและนับทั้งหมด", func(t *testing.T) {
		list, total, err := svc.List(ctx, rid, 3, 0)
		require.NoError(t, err)
		assert.EqualValues(t, 10, total)
		assert.Len(t, list, 3)
		assert.Equal(t, "ผู้รีวิว", list[0].AuthorName)
	})
}
