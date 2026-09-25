package user

import (
	"context"
	"testing"

	"jongyoung/internal/platform/testdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryUpsert(t *testing.T) {
	db := testdb.New(t)
	repo := NewRepository(db)
	ctx := context.Background()

	created, err := repo.Upsert(ctx, "sub-1", "a@b.c", "มะลิ")
	require.NoError(t, err)
	assert.NotEqual(t, [16]byte{}, created.ID)

	t.Run("sub เดิม → อัปเดตแถวเดิม ไม่สร้างแถวใหม่", func(t *testing.T) {
		updated, err := repo.Upsert(ctx, "sub-1", "new@b.c", "มะลิ วงศ์ดี")
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, "new@b.c", updated.Email)
		assert.Equal(t, "มะลิ วงศ์ดี", updated.DisplayName)
	})

	t.Run("ผู้ใช้ที่ถูก soft delete ล็อกอินใหม่ → ได้แถวใหม่ (partial unique index)", func(t *testing.T) {
		require.NoError(t, db.Delete(&User{}, "id = ?", created.ID).Error)
		again, err := repo.Upsert(ctx, "sub-1", "a@b.c", "มะลิ")
		require.NoError(t, err)
		assert.NotEqual(t, created.ID, again.ID)

		found, err := repo.FindByUID(ctx, "sub-1")
		require.NoError(t, err)
		assert.Equal(t, again.ID, found.ID, "FindByUID ต้องไม่เจอแถวที่ถูกลบ")
	})

	t.Run("ไม่มีร้าน → คืน slice ว่าง ไม่ใช่ nil", func(t *testing.T) {
		restaurants, err := repo.OwnedRestaurants(ctx, created.ID)
		require.NoError(t, err)
		assert.NotNil(t, restaurants)
		assert.Empty(t, restaurants)
	})
}
