package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceMarkRead(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, bangkok)
	user, id := uuid.New(), uuid.New()

	repo := NewMockRepository(t)
	repo.EXPECT().MarkRead(ctx, user, id, now).Return(false, nil)
	err := NewService(repo, func() time.Time { return now }).MarkRead(ctx, user, id)
	assert.ErrorIs(t, err, ErrNotFound, "ไม่มีแถวถูกแก้ = ไม่มีหรือเป็นของคนอื่น")
}
