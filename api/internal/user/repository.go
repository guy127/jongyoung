package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) FindByUID(ctx context.Context, uid string) (User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("keycloak_uid = ?", uid).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrUserNotFound
	}
	return u, err
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrUserNotFound
	}
	return u, err
}

// Upsert สร้างผู้ใช้ใหม่ หรืออัปเดต email/ชื่อ ถ้ามีอยู่แล้ว — ทำในคำสั่งเดียว
// ถ้าสอง request ของคนใหม่มาพร้อมกัน ON CONFLICT ทำให้ได้แถวเดียว ไม่ชน unique index
// predicate "WHERE deleted_at IS NULL" ต้องตรงกับ partial unique index ไม่งั้น Postgres หา index ไม่เจอ
func (r *repository) Upsert(ctx context.Context, uid, email, displayName string) (User, error) {
	var u User
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO users (keycloak_uid, email, display_name)
		VALUES (?, ?, ?)
		ON CONFLICT (keycloak_uid) WHERE deleted_at IS NULL
		DO UPDATE SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, updated_at = now()
		RETURNING *`, uid, email, displayName).Scan(&u).Error
	return u, err
}

func (r *repository) OwnedRestaurants(ctx context.Context, userID uuid.UUID) ([]OwnedRestaurant, error) {
	restaurants := []OwnedRestaurant{}
	err := r.db.WithContext(ctx).
		Table("restaurants").
		Select("id, name").
		Where("owner_id = ? AND deleted_at IS NULL", userID).
		Order("created_at").
		Scan(&restaurants).Error
	return restaurants, err
}
