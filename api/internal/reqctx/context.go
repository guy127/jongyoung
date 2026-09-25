// Package reqctx เก็บ/อ่านข้อมูลของผู้เรียกใน context ของ request
// userID ต้องมาจาก token ผ่าน middleware.JWT เท่านั้น — ห้ามรับจาก body/query
package reqctx

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const userIDKey contextKey = iota

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
