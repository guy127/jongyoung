package review

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func expectTx(repo *MockRepository) {
	repo.EXPECT().Transaction(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(Repository) error) error { return fn(repo) })
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	user, owner, rid := uuid.New(), uuid.New(), uuid.New()

	t.Run("รีวิวร้านตัวเอง → ErrOwnRestaurant", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().RestaurantOwner(ctx, rid).Return(owner, nil)
		_, err := NewService(repo).Create(ctx, owner, rid, 5, "")
		assert.ErrorIs(t, err, ErrOwnRestaurant)
	})

	t.Run("เคยรีวิวแล้ว → ErrReviewExists", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().RestaurantOwner(ctx, rid).Return(owner, nil)
		repo.EXPECT().FindOwn(ctx, rid, user).Return(Review{}, nil)
		_, err := NewService(repo).Create(ctx, user, rid, 5, "")
		assert.ErrorIs(t, err, ErrReviewExists)
	})

	t.Run("รีวิวใหม่ → บันทึกและบวกคะแนน +score, จำนวน +1", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().RestaurantOwner(ctx, rid).Return(owner, nil)
		repo.EXPECT().FindOwn(ctx, rid, user).Return(Review{}, ErrReviewNotFound)
		repo.EXPECT().Create(ctx, mock.Anything).Return(nil)
		repo.EXPECT().AdjustRating(ctx, rid, 4, 1).Return(nil)
		_, err := NewService(repo).Create(ctx, user, rid, 4, "ดี")
		assert.NoError(t, err)
	})
}

func TestUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	user, rid := uuid.New(), uuid.New()
	mine := Review{ID: uuid.New(), RestaurantID: rid, UserID: user, Score: 2}

	t.Run("แก้จาก 2 เป็น 5 → ปรับผลรวม +3 จำนวนเท่าเดิม", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().FindOwn(ctx, rid, user).Return(mine, nil)
		repo.EXPECT().Update(ctx, mock.Anything).Return(nil)
		repo.EXPECT().AdjustRating(ctx, rid, 3, 0).Return(nil)
		rv, err := NewService(repo).Update(ctx, user, rid, 5, "ดีขึ้น")
		assert.NoError(t, err)
		assert.Equal(t, 5, rv.Score)
	})

	t.Run("ยังไม่เคยรีวิวแต่เรียก PUT → ErrReviewNotFound", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().FindOwn(ctx, rid, user).Return(Review{}, ErrReviewNotFound)
		_, err := NewService(repo).Update(ctx, user, rid, 5, "")
		assert.ErrorIs(t, err, ErrReviewNotFound)
	})

	t.Run("ลบ → หักคะแนนเดิมและจำนวน 1", func(t *testing.T) {
		repo := NewMockRepository(t)
		expectTx(repo)
		repo.EXPECT().FindOwn(ctx, rid, user).Return(mine, nil)
		repo.EXPECT().Delete(ctx, mine.ID).Return(nil)
		repo.EXPECT().AdjustRating(ctx, rid, -2, -1).Return(nil)
		assert.NoError(t, NewService(repo).Delete(ctx, user, rid))
	})
}
