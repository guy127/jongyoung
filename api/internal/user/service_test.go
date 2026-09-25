package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureExists(t *testing.T) {
	ctx := context.Background()

	t.Run("ครั้งแรก (ยังไม่มี) → สร้าง", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindByUID(ctx, "sub-1").Return(User{}, ErrUserNotFound)
		repo.EXPECT().Upsert(ctx, "sub-1", "a@b.c", "มะลิ").Return(User{ID: uuid.New(), KeycloakUID: "sub-1"}, nil)

		u, err := NewService(repo).EnsureExists(ctx, "sub-1", "a@b.c", "มะลิ")
		require.NoError(t, err)
		assert.Equal(t, "sub-1", u.KeycloakUID)
	})

	t.Run("มีอยู่แล้วและข้อมูลเหมือนเดิม → ไม่เขียน DB", func(t *testing.T) {
		existing := User{ID: uuid.New(), KeycloakUID: "sub-1", Email: "a@b.c", DisplayName: "มะลิ"}
		repo := NewMockRepository(t)
		repo.EXPECT().FindByUID(ctx, "sub-1").Return(existing, nil)
		// ไม่ได้ตั้ง EXPECT ของ Upsert — ถ้าถูกเรียก mock จะทำให้เทสต์ fail

		u, err := NewService(repo).EnsureExists(ctx, "sub-1", "a@b.c", "มะลิ")
		require.NoError(t, err)
		assert.Equal(t, existing.ID, u.ID)
	})

	t.Run("ชื่อ/อีเมลใน Keycloak เปลี่ยน → อัปเดต", func(t *testing.T) {
		repo := NewMockRepository(t)
		repo.EXPECT().FindByUID(ctx, "sub-1").Return(User{KeycloakUID: "sub-1", Email: "old@b.c", DisplayName: "มะลิ"}, nil)
		repo.EXPECT().Upsert(ctx, "sub-1", "new@b.c", "มะลิ").Return(User{Email: "new@b.c"}, nil)

		u, err := NewService(repo).EnsureExists(ctx, "sub-1", "new@b.c", "มะลิ")
		require.NoError(t, err)
		assert.Equal(t, "new@b.c", u.Email)
	})
}
